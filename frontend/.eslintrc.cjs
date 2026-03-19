module.exports = {
  root: true,
  ignorePatterns: ['src/utils/geoip-example.js'],
  env: {
    browser: true,
    es2021: true,
    node: true
  },
  extends: [
    'eslint:recommended',
    'plugin:vue/vue3-recommended'
  ],
  parserOptions: {
    ecmaVersion: 'latest',
    sourceType: 'module'
  },
  plugins: ['vue', 'security'],
  overrides: [
    {
      files: ['src/utils/**/*.{js,vue}', 'src/router/**/*.{js,vue}', 'src/store/**/*.{js,vue}'],
      rules: {
        'no-unused-vars': 'error',
        'no-empty': 'error',
        'no-useless-escape': 'error',
        'no-case-declarations': 'error',
        'no-dupe-else-if': 'error'
      }
    }
  ],
  rules: {
    'vue/multi-word-component-names': 'off',
    'no-unused-vars': 'warn',
    'no-empty': 'warn',
    'no-useless-escape': 'warn',
    'no-case-declarations': 'warn',
    'no-dupe-else-if': 'warn',
    'vue/no-unused-components': 'warn',
    'vue/no-reserved-component-names': 'warn',
    'vue/no-unused-vars': 'warn',
    'vue/no-use-v-if-with-v-for': 'warn',
    'security/detect-object-injection': 'warn',
    'security/detect-non-literal-regexp': 'warn',
    'security/detect-unsafe-regex': 'error',
    'security/detect-buffer-noassert': 'error',
    'security/detect-child-process': 'error',
    'security/detect-disable-mustache-escape': 'error',
    'security/detect-eval-with-expression': 'error',
    'security/detect-no-csrf-before-method-override': 'error',
    'security/detect-non-literal-fs-filename': 'warn',
    'security/detect-non-literal-require': 'warn',
    'security/detect-possible-timing-attacks': 'warn',
    'security/detect-pseudoRandomBytes': 'error'
  }
}
