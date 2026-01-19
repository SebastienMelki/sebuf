# Create GitHub Issue Command

Create a GitHub issue from context or user-provided content with proper formatting and categorization.

## Usage Pattern
When invoked, this command will:
1. **Analyze the provided content** (from context, git changes, or user input)
2. **Determine issue type** and appropriate labels
3. **Create a well-structured GitHub issue** with relevant details

## Issue Analysis Process

1. **Content Assessment**: Examine the provided content to determine:
   - Issue type (bug, feature request, enhancement, documentation, etc.)
   - Priority level based on impact and scope
   - Relevant components/modules affected
   - Required labels and assignees

2. **Context Integration**: If no specific content is provided, analyze:
   - Recent git changes and commits
   - Current branch state and modifications
   - Existing codebase patterns and conventions

## Issue Template Structure

Create issues with the following structure:

### For Bug Reports:
```markdown
## Bug Description
[Clear description of the issue]

## Steps to Reproduce
1. [Step 1]
2. [Step 2]
3. [Expected vs Actual behavior]

## Environment
- [Relevant system/version info]

## Additional Context
[Code snippets, logs, or related information]
```

### For Feature Requests:
```markdown
## Feature Description
[Clear description of the requested feature]

## Motivation
[Why this feature is needed]

## Proposed Solution
[Detailed implementation approach if known]

## Alternatives Considered
[Other solutions that were evaluated]

## Additional Context
[Related issues, examples, or references]
```

### For Documentation Issues:
```markdown
## Documentation Issue
[What documentation needs to be updated/created]

## Current State
[What's missing or incorrect]

## Proposed Changes
[Specific improvements needed]

## Files Affected
[List of files that need updates]
```

## Labeling Strategy

Automatically apply relevant labels based on content analysis:

- **Type Labels**: `bug`, `feature`, `documentation`, `enhancement`, `refactor`
- **Priority Labels**: `priority:low`, `priority:medium`, `priority:high`, `priority:critical`
- **Component Labels**: `http-gen`, `openapi`, `validation`, `testing`
- **Status Labels**: `needs-investigation`, `ready-to-implement`, `blocked`

## Implementation Guidelines

1. **Use gh CLI**: Leverage `gh issue create` for consistent issue creation
2. **Template Selection**: Choose appropriate template based on issue type
3. **Content Parsing**: Extract relevant code snippets, error messages, or examples
4. **Link Related Issues**: Reference related PRs, issues, or commits when relevant
5. **Assign Labels**: Apply appropriate labels based on content analysis

## Example Commands

```bash
# Create issue from current git changes
gh issue create --title "[Title]" --body "$(cat <<'EOF'
[Generated issue body]
EOF
)" --label "feature,http-gen"

# Create issue with assignee
gh issue create --title "[Title]" --body "[Body]" --assignee "@me" --label "bug,priority:high"
```

## Content Sources

The command should handle input from:
- **Direct user input**: Text pasted after command invocation
- **Git context**: Current branch changes, recent commits
- **File changes**: Modified files in working directory
- **Error logs**: Captured errors or test failures
- **TODO comments**: Code TODOs that need tracking

## Quality Guidelines

Ensure created issues are:
- **Actionable**: Clear steps for resolution
- **Specific**: Focused on single concerns
- **Detailed**: Sufficient context for implementation
- **Searchable**: Good titles and relevant labels
- **Linked**: Connected to related work when applicable