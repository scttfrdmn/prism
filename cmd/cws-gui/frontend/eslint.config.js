import js from '@eslint/js';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  { ignores: ['dist', 'node_modules'] },
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      'react-refresh/only-export-components': [
        'warn',
        { allowConstantExport: true },
      ],

      // CODE QUALITY RULES (equivalent to gocyclo for Go)
      'complexity': ['error', 15],  // Maximum cyclomatic complexity of 15
      'max-lines-per-function': ['warn', { max: 100, skipBlankLines: true, skipComments: true }],
      'max-depth': ['error', 4],  // Maximum nesting depth
      'max-params': ['warn', 5],  // Maximum function parameters
      'no-console': 'warn',  // Warn on console statements

      // TypeScript specific rules for code quality
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/explicit-function-return-type': 'off',
      '@typescript-eslint/no-unused-vars': ['error', {
        argsIgnorePattern: '^_',
        varsIgnorePattern: '^_'
      }],
    },
  },
  // Temporary exception for App.tsx (large monolithic file with API client)
  // TODO: Split into separate files: API client, types, and component
  {
    files: ['src/App.tsx'],
    rules: {
      'complexity': 'off',  // Disable complexity checking for legacy monolith
      'max-lines-per-function': 'off',  // Disable function length checking
    },
  },
);
