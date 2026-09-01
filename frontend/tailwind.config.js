/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // Reference-led brand palette, softened for large UI surfaces.
        primary: {
          50: '#f4f8fa',
          100: '#e8f1f5',
          200: '#d3e5ec',
          300: '#b1d0dd',
          400: '#66a3bf',
          500: '#43789f',
          600: '#3368a0',
          700: '#2d5b8b',
          800: '#284d74',
          900: '#23384f',
          950: '#172537'
        },
        // Secondary blue/mist scale from the same reference.
        accent: {
          50: '#f4f8f7',
          100: '#e5f0ee',
          200: '#c8dfdb',
          300: '#9fc8ca',
          400: '#7ab2c1',
          500: '#66a3bf',
          600: '#4e89a6',
          700: '#3f7088',
          800: '#365c6f',
          900: '#304d5d',
          950: '#20323d'
        },
        canvas: {
          50: '#fcfbf8',
          100: '#f7f5ef',
          200: '#f2efe7',
          300: '#e7e4dc'
        },
        // Airier blue-gray surfaces keep dark mode readable without feeling heavy.
        dark: {
          50: '#f7f9fa',
          100: '#eaf0f2',
          200: '#d6e2e5',
          300: '#c9d7db',
          400: '#b7cbd1',
          500: '#a3bbc3',
          600: '#86a5af',
          700: '#60818f',
          800: '#4a7082',
          900: '#3f6477',
          950: '#34586c'
        }
      },
      fontFamily: {
        sans: [
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'PingFang SC',
          'Hiragino Sans GB',
          'Microsoft YaHei',
          'sans-serif'
        ],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace']
      },
      boxShadow: {
        glass: '0 8px 32px rgba(0, 0, 0, 0.08)',
        'glass-sm': '0 4px 16px rgba(0, 0, 0, 0.06)',
        glow: '0 0 20px rgba(102, 163, 191, 0.25)',
        'glow-lg': '0 0 40px rgba(102, 163, 191, 0.35)',
        card: '0 1px 3px rgba(0, 0, 0, 0.04), 0 1px 2px rgba(0, 0, 0, 0.06)',
        'card-hover': '0 10px 40px rgba(0, 0, 0, 0.08)',
        'inner-glow': 'inset 0 1px 0 rgba(255, 255, 255, 0.1)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(135deg, #43789f 0%, #3368a0 100%)',
        'gradient-dark': 'linear-gradient(135deg, #4a7082 0%, #34586c 100%)',
        'gradient-glass':
          'linear-gradient(135deg, rgba(255,255,255,0.1) 0%, rgba(255,255,255,0.05) 100%)',
        'mesh-gradient':
          'radial-gradient(at 40% 20%, rgba(102, 163, 191, 0.12) 0px, transparent 50%), radial-gradient(at 80% 0%, rgba(139, 185, 203, 0.12) 0px, transparent 50%), radial-gradient(at 0% 50%, rgba(200, 223, 219, 0.24) 0px, transparent 50%)'
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'glow 2s ease-in-out infinite alternate'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        glow: {
          '0%': { boxShadow: '0 0 20px rgba(102, 163, 191, 0.25)' },
          '100%': { boxShadow: '0 0 30px rgba(102, 163, 191, 0.4)' }
        }
      },
      backdropBlur: {
        xs: '2px'
      },
      borderRadius: {
        '4xl': '2rem'
      }
    }
  },
  plugins: []
}
