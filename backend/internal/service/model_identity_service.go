package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type ModelIdentityService struct {
	repo      IdentityRepository
	apiKeys   *APIKeyService
	accounts  AccountRepository
	engineURL string
	remoteURL string
	publicURL string
	client    *http.Client
}

func NewModelIdentityService(repo IdentityRepository, apiKeys *APIKeyService, accounts AccountRepository) *ModelIdentityService {
	return &ModelIdentityService{repo: repo, apiKeys: apiKeys, accounts: accounts,
		engineURL: strings.TrimRight(os.Getenv("MODEL_IDENTITY_ENGINE_URL"), "/"),
		remoteURL: strings.TrimRight(os.Getenv("MODEL_IDENTITY_REMOTE_API_URL"), "/"),
		publicURL: strings.TrimRight(os.Getenv("MODEL_IDENTITY_PUBLIC_BASE_URL"), "/"),
		client:    &http.Client{Timeout: 10 * time.Second}}
}
func (s *ModelIdentityService) EngineCommit() string {
	if s.remoteURL != "" {
		return "bazaarlink-online"
	}
	return IdentityEngineCommit
}

func (s *ModelIdentityService) Models(ctx context.Context) ([]IdentityModel, error) {
	if s.remoteURL != "" {
		return s.remoteModels(ctx)
	}
	if s.engineURL == "" {
		return nil, errors.New("MODEL_IDENTITY_ENGINE_URL is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.engineURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	res, err := s.client.Do(req)
	if err != nil {
		return nil, errors.New("identity engine unavailable")
	}
	defer func() { _ = res.Body.Close() }()
	var catalog struct {
		Commit string          `json:"engine_commit"`
		Models []IdentityModel `json:"models"`
	}
	if res.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&catalog) != nil || catalog.Commit != IdentityEngineCommit {
		return nil, errors.New("identity engine version mismatch or invalid catalog")
	}
	return catalog.Models, nil
}

func (s *ModelIdentityService) remoteModels(ctx context.Context) ([]IdentityModel, error) {
	u, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteBaselinesURL(s.remoteURL), nil)
	if err != nil {
		return nil, err
	}
	res, err := s.client.Do(u)
	if err != nil {
		return nil, errors.New("BazaarLink Probe API unavailable")
	}
	defer func() { _ = res.Body.Close() }()
	var payload struct {
		Models []string `json:"models"`
	}
	if res.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&payload) != nil {
		return nil, errors.New("invalid BazaarLink baseline catalog")
	}
	models := make([]IdentityModel, 0, len(payload.Models))
	for _, id := range payload.Models {
		family := id
		if i := strings.IndexByte(family, '/'); i >= 0 {
			family = family[:i]
		}
		models = append(models, IdentityModel{ID: id, Name: id, Family: family})
	}
	return models, nil
}

func remoteBaselinesURL(remote string) string {
	u, err := http.NewRequest(http.MethodGet, remote, nil)
	if err != nil {
		return strings.TrimRight(remote, "/") + "/../baselines"
	}
	u.URL.Path = strings.TrimSuffix(u.URL.Path, "/run") + "/baselines"
	u.URL.RawQuery = ""
	return u.URL.String()
}
func identityAccountInGroup(a *Account, g int64) bool {
	for _, x := range a.AccountGroups {
		if x.GroupID == g {
			return true
		}
	}
	for _, x := range a.GroupIDs {
		if x == g {
			return true
		}
	}
	return false
}
func (s *ModelIdentityService) validateBinding(ctx context.Context, accountID, userID, groupID int64) (*Account, *Group, error) {
	a, e := s.accounts.GetByID(ctx, accountID)
	if e != nil {
		return nil, nil, errors.New("configuration error: target account not found")
	}
	g, e := s.apiKeys.groupRepo.GetByID(ctx, groupID)
	if e != nil {
		return nil, nil, errors.New("configuration error: billing group not found")
	}
	if a == nil || g == nil || g.Status != StatusActive || !identityAccountInGroup(a, groupID) || a.Platform != g.Platform || g.ClaudeCodeOnly {
		return nil, nil, errors.New("configuration error: choose a compatible group containing the target account")
	}
	u, e := s.apiKeys.userRepo.GetByID(ctx, userID)
	if e != nil || u == nil || !u.IsActive() {
		return nil, nil, errors.New("configuration error: active test user required")
	}
	if !s.apiKeys.canUserBindGroup(ctx, u, g) {
		return nil, nil, ErrGroupNotAllowed
	}
	return a, g, nil
}
func (s *ModelIdentityService) Configure(ctx context.Context, accountID, userID, groupID int64) (*IdentityConfig, error) {
	a, g, e := s.validateBinding(ctx, accountID, userID, groupID)
	if e != nil {
		return nil, e
	}
	name := fmt.Sprintf("模型身份测试专用｜%s｜%s (#%d)", g.Name, a.Name, a.ID)
	if len([]rune(name)) > 100 {
		return nil, errors.New("dedicated Key name exceeds 100 characters; shorten the account or group name")
	}
	previous, previousErr := s.repo.Config(ctx, accountID)
	var token string
	if previousErr != nil {
		if !errors.Is(previousErr, sql.ErrNoRows) {
			return nil, previousErr
		}
		if e = s.apiKeys.checkAPIKeyCreateLimits(ctx, userID); e != nil {
			return nil, e
		}
		token, e = s.apiKeys.GenerateKey()
		if e != nil {
			return nil, e
		}
	}
	var previousKey *APIKey
	if previousErr == nil && previous != nil {
		previousKey, _ = s.apiKeys.GetByID(ctx, previous.APIKeyID)
	}
	c := &IdentityConfig{AccountID: accountID, UserID: userID, GroupID: groupID}
	if e = s.repo.Configure(ctx, c, name, token); e != nil {
		return nil, e
	}
	if previousKey != nil {
		s.apiKeys.InvalidateAuthCacheByKey(ctx, previousKey.Key)
	}
	if k, e := s.apiKeys.GetByID(ctx, c.APIKeyID); e == nil {
		s.apiKeys.InvalidateAuthCacheByKey(ctx, k.Key)
	}
	return c, nil
}
func (s *ModelIdentityService) ValidateRun(ctx context.Context, r *IdentityRun) (*APIKey, error) {
	if _, _, e := s.validateBinding(ctx, r.AccountID, r.UserID, r.GroupID); e != nil {
		return nil, e
	}
	k, e := s.apiKeys.GetByID(ctx, r.APIKeyID)
	if e != nil || k == nil || k.UserID != r.UserID || k.GroupID == nil || *k.GroupID != r.GroupID || !k.IsActive() || k.IsExpired() || k.IsQuotaExhausted() {
		return nil, errors.New("configuration error: dedicated API Key was deleted, disabled, rebound, expired or exhausted; repair the existing Key")
	}
	return k, nil
}
func (s *ModelIdentityService) Config(ctx context.Context, id int64) (*IdentityConfig, error) {
	c, err := s.repo.Config(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err := s.ValidateRun(ctx, &IdentityRun{AccountID: c.AccountID, UserID: c.UserID, GroupID: c.GroupID, APIKeyID: c.APIKeyID}); err != nil {
		c.ConfigurationError = err.Error()
	}
	return c, nil
}

// AuthorizeRemote binds the permanent test Key to a currently leased run.
// The public callback never accepts a client-selected account or group.
func (s *ModelIdentityService) AuthorizeRemote(ctx context.Context, id int64, credential string) (*IdentityRun, *APIKey, error) {
	run, err := s.repo.Run(ctx, id)
	if err != nil || run == nil || run.Status != "running" || run.StartedAt == nil || !time.Now().Before(run.StartedAt.Add(20*time.Minute)) {
		return nil, nil, errors.New("detection run inactive or expired")
	}
	key, err := s.apiKeys.GetByID(ctx, run.APIKeyID)
	if err != nil || key == nil || len(credential) == 0 || subtle.ConstantTimeCompare([]byte(HashIdentityToken(key.Key)), []byte(HashIdentityToken(credential))) != 1 {
		return nil, nil, errors.New("invalid dedicated test Key")
	}
	if _, err := s.ValidateRun(ctx, run); err != nil {
		return run, key, err
	}
	return run, key, nil
}
func (s *ModelIdentityService) Plans(ctx context.Context, id int64) ([]IdentityPlan, error) {
	return s.repo.Plans(ctx, id)
}
func (s *ModelIdentityService) SavePlan(ctx context.Context, p *IdentityPlan) error {
	if p.IntervalMinutes == 0 {
		p.IntervalMinutes = 120
	}
	if p.IntervalMinutes < 15 || p.IntervalMinutes > 10080 {
		return errors.New("interval must be 15 minutes to 7 days")
	}
	if p.ID != 0 {
		existing, err := s.repo.Plan(ctx, p.ID)
		if err != nil {
			return err
		}
		p.AccountID = existing.AccountID
		// Pausing an unchanged plan must work even if its Key or the catalog is unavailable.
		if !p.Enabled && p.RequestModel == existing.RequestModel && p.ExpectedModel == existing.ExpectedModel && p.IntervalMinutes == existing.IntervalMinutes {
			return s.repo.SavePlan(ctx, p, time.Now())
		}
	}
	models, e := s.Models(ctx)
	if e != nil {
		return e
	}
	valid := false
	for _, m := range models {
		if m.ID == p.ExpectedModel {
			valid = true
		}
	}
	if !valid {
		return errors.New("expected model is not supported by the current identity baseline catalog")
	}
	if strings.TrimSpace(p.RequestModel) == "" || len(p.RequestModel) > 256 {
		return errors.New("request model is required and must be at most 256 bytes")
	}
	c, e := s.repo.Config(ctx, p.AccountID)
	if e != nil {
		return errors.New("save account test configuration first")
	}
	if _, e = s.ValidateRun(ctx, &IdentityRun{AccountID: c.AccountID, UserID: c.UserID, GroupID: c.GroupID, APIKeyID: c.APIKeyID}); e != nil {
		return e
	}
	return s.repo.SavePlan(ctx, p, time.Now())
}
func (s *ModelIdentityService) DeletePlan(ctx context.Context, id int64) error {
	return s.repo.DeletePlan(ctx, id)
}
func (s *ModelIdentityService) Enqueue(ctx context.Context, id int64) (*IdentityRun, error) {
	if s.engineURL == "" && s.remoteURL == "" {
		return nil, errors.New("identity engine is not configured")
	}
	p, e := s.repo.Plan(ctx, id)
	if e != nil {
		return nil, e
	}
	c, e := s.repo.Config(ctx, p.AccountID)
	if e != nil {
		return nil, e
	}
	if _, e = s.ValidateRun(ctx, &IdentityRun{AccountID: c.AccountID, UserID: c.UserID, GroupID: c.GroupID, APIKeyID: c.APIKeyID}); e != nil {
		return nil, e
	}
	return s.repo.Enqueue(ctx, id)
}
func (s *ModelIdentityService) Cancel(ctx context.Context, id int64) error {
	return s.repo.Cancel(ctx, id)
}
func (s *ModelIdentityService) Run(ctx context.Context, id int64) (*IdentityRun, error) {
	return s.repo.Run(ctx, id)
}
func (s *ModelIdentityService) History(ctx context.Context, id int64) ([]IdentityRun, error) {
	return s.repo.History(ctx, id)
}
func (s *ModelIdentityService) AppendProbe(ctx context.Context, id int64, probe json.RawMessage) error {
	return s.repo.AppendProbe(ctx, id, probe)
}
func (s *ModelIdentityService) ProbeSlot(ctx context.Context, id int64, acquire bool) error {
	return s.repo.ProbeSlot(ctx, id, acquire)
}
func HashIdentityToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
func newIdentityToken() (string, error) {
	b := make([]byte, 32)
	_, e := rand.Read(b)
	return hex.EncodeToString(b), e
}
func (s *ModelIdentityService) Authorize(ctx context.Context, id int64, token string) (*IdentityRun, error) {
	if len(token) != 64 {
		return nil, errors.New("invalid detection capability")
	}
	return s.repo.Authorize(ctx, id, HashIdentityToken(token), time.Now())
}
func (s *ModelIdentityService) Finish(ctx context.Context, id int64, status string, report any) error {
	b, e := json.Marshal(report)
	if e != nil {
		return e
	}
	return s.repo.Finish(ctx, id, status, b)
}
