module.exports = {
  env: {
    browser: true,
    es2021: true,
    node: true,
  },
  extends: [
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:react-hooks/recommended',
    'prettier',
  ],
  ignorePatterns: ['dist', '.eslintrc.cjs', 'node_modules'],
  parser: '@typescript-eslint/parser',
  plugins: ['react-hooks', 'react-refresh', '@typescript-eslint'],
  rules: {
    'react-refresh/only-export-components': [
      'warn',
      {
        allowConstantExport: true,
      },
    ],
    'react-hooks/static-components': 'off',
    'react-hooks/purity': 'off',
    'react-hooks/set-state-in-effect': 'off',
    'react-hooks/immutability': 'off',
    '@typescript-eslint/no-explicit-any': 'warn',
    "@typescript-eslint/no-unused-vars": "warn",
    "prefer-const": "warn"
  }
}
