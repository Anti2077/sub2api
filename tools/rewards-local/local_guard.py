"""Bind gateway listener, its config/cwd/connections and Compose fixture before writes."""
from pathlib import Path
import json
import subprocess

ROOT = Path('/private/tmp/rewards-local')
COMPOSE = ['docker', 'compose', '-p', 'sub2api-rewards-local', '-f', str(ROOT / 'compose.yaml')]


def sql(q):
    return subprocess.check_output(COMPOSE + ['exec', '-T', 'postgres', 'psql', '-U', 'rewards_test', '-d', 'rewards_test', '-At', '-v', 'ON_ERROR_STOP=1', '-c', q], text=True).strip()


def guard():
    pid = subprocess.check_output(['lsof', '-t', '-iTCP:18086', '-sTCP:LISTEN'], text=True).strip()
    assert pid.isdigit(), 'Expected exactly one local gateway'
    cwd = subprocess.check_output(['lsof', '-a', '-p', pid, '-d', 'cwd', '-Fn'], text=True)
    assert '\nn/private/tmp/rewards-local/data\n' in cwd
    connections = subprocess.check_output(['lsof', '-a', '-p', pid, '-iTCP:15486', '-nP'], text=True)
    assert '->127.0.0.1:15486 (ESTABLISHED)' in connections
    cfg, section = {}, ''
    for line in (ROOT / 'data/config.yaml').read_text().splitlines():
        if not line.startswith(' '):
            section = line.split(':')[0]
        elif ':' in line:
            k, v = line.strip().split(':', 1)
            if k in ['host', 'port', 'user', 'dbname']:
                cfg[section + '.' + k] = v.strip()
    assert all(cfg[k] == v for k, v in {
        'server.host': '127.0.0.1', 'server.port': '18086',
        'database.host': '127.0.0.1', 'database.port': '15486',
        'database.user': 'rewards_test', 'database.dbname': 'rewards_test',
        'redis.host': '127.0.0.1', 'redis.port': '16386'}.items())
    rows = [json.loads(v) for v in subprocess.check_output(COMPOSE + ['ps', '--format', 'json'], text=True).splitlines()]
    pg = next(v for v in rows if v['Service'] == 'postgres')
    assert pg['Project'] == 'sub2api-rewards-local'
    assert pg['Publishers'] == [{'URL': '127.0.0.1', 'TargetPort': 5432, 'PublishedPort': 15486, 'Protocol': 'tcp'}]
    assert sql("SELECT current_database()||'|'||current_user") == 'rewards_test|rewards_test'
    assert sql("SELECT COUNT(*) FROM users WHERE email NOT LIKE '%@rewards.test'") == '0'


if __name__ == '__main__':
    guard()
    print('Local gateway/config/connections/Compose/database fixture identity verified.')
