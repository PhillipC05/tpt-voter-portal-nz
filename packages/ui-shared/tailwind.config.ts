import type { Config } from 'tailwindcss'

const config: Config = {
  content: [
    './src/**/*.{ts,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        // NZ Government brand palette (NZ Government Design System)
        // https://design-system.digital.govt.nz/design/colour/
        nzgds: {
          blue: '#00538C',       // Primary action / RealMe brand
          'blue-dark': '#004070',
          'blue-light': '#E8F0F7',
          teal: '#006272',       // Secondary brand
          'teal-light': '#E5F2F4',
          green: '#007B40',      // Success / Verified status
          'green-light': '#E5F2EB',
          red: '#C44000',        // Error / destructive
          'red-light': '#FAEAE3',
          orange: '#E07500',     // Warning
          'orange-light': '#FDF4E3',
          grey: {
            50:  '#F8F8F8',
            100: '#EEEEEE',
            200: '#DEDEDE',
            300: '#BCBCBC',
            400: '#9A9A9A',
            500: '#787878',
            600: '#565656',
            700: '#444444',
            800: '#333333',
            900: '#1A1A1A',
          },
        },
      },
      fontFamily: {
        sans: ['Public Sans', 'Noto Sans', 'Arial', 'sans-serif'],
      },
      borderRadius: {
        sm: '2px',
        DEFAULT: '4px',
        md: '4px',
        lg: '8px',
      },
      spacing: {
        // 4px base grid
        0.5: '2px',
        1:   '4px',
        1.5: '6px',
        2:   '8px',
        3:   '12px',
        4:   '16px',
        6:   '24px',
        8:   '32px',
        10:  '40px',
        12:  '48px',
        16:  '64px',
      },
    },
  },
  plugins: [],
}

export default config
