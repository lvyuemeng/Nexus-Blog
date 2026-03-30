+++
date = '2025-03-24T22:53:47+08:00'
draft = false
title = 'Deploy WorkFlow(2)'
description = 'Guide to collaboration workflows: branch rules, pull requests, and interactive rebase for code review.'
tags = ["Deploy"]
author = ["nostalgia"]
+++

## Collaboration

### Rules

You can set rules for collaboration and to prevent safety issues.

Usually, you want to prevent someone from modifying the `main` branch arbitrarily and restrict all push behavior after a careful check on PRs.

- Settings
  - Rules
    - restrict creations
    - restrict update
    - restrict deletions
    - block force pushes

Here are some simple block rules you may want to set.

### Pull Request

You should create a new branch to avoid conflicts with your modifications, then create a new pull request to submit changes.

```bash
git checkout -b <new-branch-name>
```

Here `-b` means a new branch.

After modification, you can push it.

```bash
git push origin <branch-name>
```

Then create `New Pull Request` in the github repo, github will detect your remote push branch and suggest the pr format.

### Code Review

Usually, a branch may contain many modifications and diverge. You may want to keep it neat and readable. So, we can do the following:

```bash
git rebase -i HEAD~<number>
```

- `rebase` means merge the specified branch in linearization rather than indicating a commit; currently we don't specify any branch, so it means `rebase` itself.
- `-i` means interactive
- `HEAD~<number>` means to include your specified branch range from `HEAD` to its previous commits until `<number>`.

Now git will open editor and show the content of commits.
Here are verbs you should know:
- pick(p): Include this commit in the final history as-is.
- reword(r): Include this commit, but edit its commit message.
- edit(e): Include this commit, but pause the rebase process to allow you to amend the commit.
- squash(s): Include this commit, but meld it into the previous commit.
- fixup(f): Like squash, but discard this commit message.
- drop(d): Remove this commit from the final history.

The above reference of **commit** includes its content and message, so if you `drop` a commit, it will remove all code it modifies and its message!

## Summary

Here is all you need for a simple static blog deployment, including some git knowledge. You can refer to the [Git Book](https://git-scm.com/book/en/v2) if you want to learn more!