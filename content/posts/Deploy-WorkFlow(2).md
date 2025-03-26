+++
date = '2025-03-24T22:53:47+08:00'
draft = false
title = 'Deploy WorkFlow(2)'
tag = ["Deploy"]
author = ["nostalgia"]
+++

## Collaboration

### Rules

You could set rules for collaboration and preventing safety issues.

Usually, you want prevent someone modify `main` branch arbitrarily and restrict all push behavior after a carefully check on pr.

- Settings
  - Rules
    - restrict creations
    - restrict update
    - restrict deletions
    - block force pushes
	
Here are some simple block rules you want to set.

### Pull Request

You should create a new branch to circumvent awkward situation for your modification, you can create a new pull request to do this in ease.

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

Usually, a branch may contains many modification and diverge, you may want to keep it neat and readable. So, we can do the following:

```bash
git rebase -i HEAD~<number>
```

- `rebase` means merge the specified branch in linearization rather indicate a commit, currently we don't specify any branch, it means `rebase` itself.
- '-i' means interactive
- `HEAD~<number>` means to include your specify branch range from `HEAD` to its previous commits until `<number>` your specified.

Now git will open editor and show the content of commits.
Here are verbs you should know:
- pick(p): Include this commit in the final history as-is.
- reword(r): Include this commit, but edit its commit message.
- edit(e):Include this commit, but pause the rebase process to allow you to amend the commit.
- squash(s): Include this commit, but meld it into the previous commit.
- fixup(f): Like squash, but discard this commit message.
- drop(d): Remove this commit from the final history.

The Above reference of **commit** include its content and message, so if you `drop` commit, it will remove all code it modifies and message!

## Summary

Here are all you want to do for a simple static blog deployment, including a bit of knowledges about git. You can refer [Git Book](https://git-scm.com/book/en/v2) if you want! Thanks for your reading!