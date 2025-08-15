# Smart Git Commit Instructions

When committing code changes, follow these guidelines to create intelligent, organized commits:

## Commit Strategy

1. **Analyze modified files** and group them by relationship and purpose
2. **Create separate commits** for logically distinct changes
3. **Use conventional commit format**: `type(scope): description`

## File Grouping Rules

Group files into separate commits based on:

- **Tests**: Files ending in `.test.js`, `.spec.ts`, `_test.py`, etc.
- **Documentation**: README files, `.md` files, changelogs
- **Dependencies**: `package.json`, `requirements.txt`, `Cargo.toml`, etc.
- **Configuration**: Config files, `.env`, webpack configs, etc.
- **Styles**: CSS, SCSS, styling files
- **Features**: Related implementation files (components, utilities, etc.)
- **Build/CI**: Dockerfile, GitHub Actions, build scripts

## Conventional Commit Types

- `feat`: New features or functionality
- `fix`: Bug fixes
- `docs`: Documentation changes
- `style`: Code style changes (formatting, no logic change)
- `refactor`: Code refactoring without feature changes
- `test`: Adding or updating tests
- `build`: Build system or dependency changes
- `ci`: CI/CD configuration changes
- `chore`: Other maintenance tasks

## Example Workflow

For a change that includes:
- `src/components/Button.tsx` (new component)
- `src/components/Button.test.tsx` (tests)
- `README.md` (documentation)
- `package.json` (new dependency)

Create 4 separate commits:
1. `feat(components): add Button component`
2. `test(components): add Button component tests`
3. `docs: update README with Button component usage`
4. `build: add new UI dependency`

## Implementation Instructions

1. **Stage files by group**: Use `git add` for each logical group
2. **Create meaningful commit messages**: Be specific about what changed
3. **Keep commits focused**: Each commit should have a single purpose
4. **Consider revert scenarios**: Structure commits so individual changes can be reverted safely

## Commit Message Guidelines

- Use present tense ("add feature" not "added feature")
- Keep first line under 50 characters
- Be descriptive but concise
- Include scope when relevant: `feat(auth): add login validation`
- Add body with more details if needed (after blank line)

## Example Commands

```bash
# Group 1: Feature implementation
git add src/components/UserProfile.tsx src/utils/userHelpers.ts
git commit -m "feat(user): add user profile component with helper utilities"

# Group 2: Tests
git add src/components/UserProfile.test.tsx src/utils/userHelpers.test.ts
git commit -m "test(user): add comprehensive tests for user profile feature"

# Group 3: Documentation
git add README.md docs/user-guide.md
git commit -m "docs: update documentation for user profile feature"
```

## Benefits

- **Easier code review**: Each commit focuses on one logical change
- **Better git history**: Clear progression of changes
- **Safer reverts**: Can revert specific functionality without affecting other changes
- **Clearer debugging**: git bisect works more effectively
- **Team collaboration**: Easier to understand what changed and why
