export default {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [2, 'always', ['feat', 'fix', 'chore', 'ci', 'docs', 'refactor', 'test']],
    'scope-empty': [0, 'never'],
    'scope-enum': [2, 'always', ['repo', 'deps', 'sync', 'validate', 'update']],
    'subject-case': [2, 'never', ['sentence-case']],
    'subject-empty': [2, 'never']
  },
};
