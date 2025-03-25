# Contribution Guidelines

We welcome and appreciate contributions from the community! Whether you want to fix a typo, improve code, add new features, or submit guest posts, please follow these guidelines to ensure a smooth collaboration process.

## How to Contribute

### 1. Contributing Code / Bug Fixes
- Report bugs or suggest features by [opening an issue](https://github.com/lvyuemeng/opencamp-blog/issues/new)
- Fork the repository
- Create a feature/bugfix branch:  
  `git checkout -b feature/your-feature-name` or `git checkout -b fix/your-bugfix-name`
- Commit your changes with descriptive messages
- Push to your branch: `git push origin your-branch-name`
- Open a Pull Request (PR) against the `main` branch

### 2. Contributing Articles
- Create a new branch for each article: `git checkout -b new_branch`
- All articles should be written in Markdown (.md)
- Place files in `content/posts/` directory
- Follow naming convention: `your_article_title_YYYY-MM-DD.md`
- Include proper front matter:
  ```markdown
  ---
  title = "Your Article Title"
  date = YYYY-MM-DD
  author = Your Name (or pseudonym)
  tags = [tag1, tag2]
  categories = [category1]
  ---
  
  Your article content goes here...
  ```
- Commit your changes and push to your branch
- Open a Pull Request (PR) against the `main` branch