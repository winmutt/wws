# Contributing to WWS

## Welcome!

Thank you for your interest in contributing to WWS (Winmutt's Work Spaces). This document provides guidelines and instructions for contributing.

## Code of Conduct

- Be respectful and inclusive
- Provide constructive feedback
- Accept constructive criticism
- Focus on what's best for the community

## How to Contribute

### Reporting Bugs

1. **Check existing issues** before creating a new one
2. **Use bug report template**
3. **Include:**
   - Description of the bug
   - Steps to reproduce
   - Expected behavior
   - Actual behavior
   - Environment details
   - Screenshots if applicable

### Suggesting Features

1. **Check existing feature requests**
2. **Use feature request template**
3. **Include:**
   - Problem description
   - Proposed solution
   - Use cases
   - Alternative solutions considered

### Pull Requests

1. **Fork the repository**
2. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```
3. **Make your changes**
4. **Write tests** for new functionality
5. **Update documentation** as needed
6. **Submit a PR**

## Development Workflow

### Branch Naming

```
feature/<component>/<description>
fix/<component>/<description>
docs/<component>/<description>
```

Examples:
- `feature/auth/github-oauth`
- `fix/workspace/provisioning-error`
- `docs/api/endpoints`

### Commit Messages

Follow conventional commits:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Formatting
- `refactor`: Code restructuring
- `test`: Tests
- `chore`: Maintenance

Examples:
```
feat(auth): Add GitHub OAuth2 support
fix(workspace): Fix provisioning timeout error
docs(api): Update endpoint documentation
```

### Code Style

#### Go

- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Run `gofmt` before committing
- Use `golangci-lint` for linting

```bash
# Format code
gofmt -w .

# Lint code
golangci-lint run
```

#### TypeScript/React

- Follow ESLint configuration
- Use functional components with hooks
- Include TypeScript interfaces for props

```bash
# Lint code
npm run lint

# Format code
npm run format
```

## Testing

### Running Tests

```bash
# Backend tests
go test ./api/... -v

# Frontend tests
cd web && npm test

# Integration tests
cd tests && ./workspace_lifecycle_test.sh

# E2E tests
cd tests/e2e && ./run_phase25_tests.sh
```

### Test Requirements

- Unit tests for new features
- Integration tests for API changes
- E2E tests for critical workflows
- Minimum 80% code coverage

## Documentation

### Updating Documentation

- Update README.md for major changes
- Update API documentation for API changes
- Add examples for new features
- Update architecture docs for design changes

### Documentation Files

- `README.md`: Project overview
- `docs/ARCHITECTURE.md`: System architecture
- `docs/REST_API.md`: API documentation
- `docs/DEVELOPER_SETUP.md`: Development setup
- `docs/DEPLOYMENT.md`: Deployment guide
- `docs/SECURITY.md`: Security best practices

## Review Process

### Code Review

1. **Automated Checks**
   - CI/CD pipeline passes
   - Tests pass
   - Code coverage maintained

2. **Human Review**
   - At least 1 reviewer for bug fixes
   - At least 2 reviewers for features
   - Security review for security-related changes

### Review Criteria

- Code quality and readability
- Test coverage
- Documentation updates
- Performance considerations
- Security implications

## Getting Help

### Communication Channels

- **GitHub Issues**: Bug reports, feature requests
- **GitHub Discussions**: General questions, discussions
- **Email**: dev@winmutt.github.com (for sensitive topics)

### Before Asking

- Check existing issues and documentation
- Search for similar questions
- Provide context and details

## Release Process

### Versioning

We use [Semantic Versioning](https://semver.org/):

- `MAJOR` for breaking changes
- `MINOR` for new features (backward compatible)
- `PATCH` for bug fixes

### Releases

1. Create release branch
2. Update version numbers
3. Update CHANGELOG
4. Create tag
5. Build and publish artifacts

## Attribution

This CONTRIBUTING.md is adapted from open source best practices.

## Questions?

If you have questions about contributing, please open a discussion or email us at dev@winmutt.github.com.

Thank you for contributing to WWS!
