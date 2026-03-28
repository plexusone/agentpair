# Git Worktrees

AgentPair can create isolated git worktrees for each run, keeping your main branch clean.

## What is a Worktree?

A git worktree is a linked working directory that shares the same repository history but has its own working tree and index. Changes in the worktree don't affect your main checkout until merged.

```
my-repo/                     # Main checkout (main branch)
├── .git/
├── src/
└── ...

.agentpair-worktree-1/       # Worktree (agentpair/run-1 branch)
├── .git                     # File pointing to main .git
├── src/                     # Separate working tree
└── ...
```

## Prerequisites

- Git repository (initialized)
- At least one commit in the repository

## Usage

```bash
agentpair --worktree --prompt "Experimental feature X"
```

AgentPair will:

1. Create a new branch: `agentpair/run-{id}`
2. Create a worktree: `.agentpair-worktree-{id}` (in parent directory)
3. Run agents in the worktree
4. Clean up worktree on completion (branch remains)

## Directory Structure

After creating a worktree:

```
/home/user/projects/
├── my-repo/                        # Your main checkout
│   ├── .git/
│   │   └── worktrees/
│   │       └── .agentpair-worktree-1/
│   └── src/
└── .agentpair-worktree-1/          # AgentPair worktree
    ├── .git                        # Points to main .git
    └── src/                        # Isolated working tree
```

## Branch Naming

| Run ID | Branch Name |
|--------|-------------|
| 1 | `agentpair/run-1` |
| 2 | `agentpair/run-2` |
| 42 | `agentpair/run-42` |

## Workflow

### 1. Start with Worktree

```bash
cd ~/projects/my-repo
agentpair --worktree --prompt "Add new authentication system"
```

### 2. Agents Work in Isolation

All changes happen in `.agentpair-worktree-1/`, not your main checkout.

### 3. Review Changes

After completion, review the branch:

```bash
# See the branch
git branch -a | grep agentpair

# Check out the branch
git checkout agentpair/run-1

# Or view diff
git diff main..agentpair/run-1
```

### 4. Merge If Satisfied

```bash
git checkout main
git merge agentpair/run-1

# Or create a PR
git push origin agentpair/run-1
gh pr create --base main --head agentpair/run-1
```

### 5. Cleanup

The worktree is automatically removed on completion. To clean up the branch:

```bash
git branch -d agentpair/run-1
```

## With tmux

Combine worktree with tmux:

```bash
agentpair --worktree --tmux --prompt "Risky refactoring"
```

## Manual Worktree Management

### List Worktrees

```bash
git worktree list
```

Output:

```
/home/user/my-repo                  abc1234 [main]
/home/user/.agentpair-worktree-1    def5678 [agentpair/run-1]
```

### Remove Worktree Manually

If a worktree wasn't cleaned up:

```bash
git worktree remove .agentpair-worktree-1
```

### Prune Stale Worktrees

```bash
git worktree prune
```

## Example Use Cases

### Experimental Features

```bash
agentpair --worktree --prompt "Try implementing feature with new architecture"
```

If it works: merge. If not: delete the branch.

### Parallel Development

Run multiple AgentPair sessions on different features:

```bash
# Terminal 1
agentpair --worktree --prompt "Add user management"

# Terminal 2
agentpair --worktree --prompt "Add payment integration"
```

Each gets its own worktree and branch.

### Safe Refactoring

```bash
agentpair --worktree --prompt "Refactor database layer" \
  --proof "All tests must pass"
```

Main branch stays untouched until you're satisfied.

## Resuming with Worktree

When resuming a run that used worktree:

```bash
agentpair --run-id 1
```

AgentPair will:

1. Check if the worktree still exists
2. Re-enter or recreate the matching worktree
3. Continue work on the same branch

## Troubleshooting

### "Not a git repository"

Worktree requires an initialized git repo:

```bash
git init
git add .
git commit -m "Initial commit"
```

### Worktree Already Exists

If a worktree wasn't cleaned up:

```bash
# Remove the stale worktree
git worktree remove .agentpair-worktree-1

# Or force remove
git worktree remove --force .agentpair-worktree-1
```

### Branch Conflicts

If the branch name already exists:

```bash
# Delete the old branch
git branch -D agentpair/run-1

# Then retry
agentpair --worktree --prompt "Task"
```

## Best Practices

1. **Use for risky changes** — Worktrees isolate experimental work
2. **Review before merging** — Check the branch thoroughly
3. **Clean up branches** — Delete merged or abandoned branches
4. **Combine with proof** — Ensure tests pass before considering merge

## Next Steps

- [Resuming Runs](resuming.md) — Continue interrupted sessions
- [tmux Layout](tmux.md) — Visual monitoring
