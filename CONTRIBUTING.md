# Contributing to live-webrtc-go

Thank you for your interest in contributing to live-webrtc-go! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct. Please read it before contributing.

## How to Contribute

### Reporting Issues

1. **Search existing issues** first to avoid duplicates
2. **Use the issue template** when creating new issues
3. **Provide detailed information** including:
   - Steps to reproduce the problem
   - Expected vs actual behavior
   - Environment details (OS, Go version, etc.)
   - Error logs and stack traces

### Suggesting Features

1. **Check existing issues** for similar feature requests
2. **Explain the use case** and why the feature would be valuable
3. **Consider implementation complexity** and maintenance burden
4. **Be open to discussion** and alternative approaches

### Code Contributions

#### Development Setup

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/live-webrtc-go.git
   cd live-webrtc-go
   ```

3. **Install dependencies**:
   ```bash
   go mod download
   ```

4. **Install development tools**:
   ```bash
   make install-tools
   ```

#### Making Changes

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following our coding standards:
   - Follow Go best practices and idioms
   - Write comprehensive tests for new functionality
   - Update documentation as needed
   - Ensure all tests pass

3. **Test your changes**:
   ```bash
   make test          # Run all tests
   make lint          # Run linters
   make security      # Run security checks
   make coverage      # Generate coverage report
   ```

4. **Commit your changes**:
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

   Follow [Conventional Commits](https://www.conventionalcommits.org/) specification:
   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation changes
   - `test:` for test additions/changes
   - `refactor:` for code refactoring
   - `perf:` for performance improvements
   - `security:` for security-related changes

#### Submitting Changes

1. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create a Pull Request** on GitHub:
   - Use the PR template
   - Provide a clear description of changes
   - Reference any related issues
   - Ensure all CI checks pass

3. **Address review feedback** promptly and professionally

### Testing Guidelines

#### Unit Tests

- Write unit tests for all new functionality
- Aim for >80% code coverage
- Use table-driven tests where appropriate
- Mock external dependencies

#### Integration Tests

- Test API endpoints and interactions
- Verify WebRTC functionality
- Test authentication and authorization
- Validate error handling

#### Performance Tests

- Benchmark critical code paths
- Test under load conditions
- Monitor memory usage and leaks
- Validate response times

#### Security Tests

- Test authentication bypass attempts
- Validate input sanitization
- Check for injection vulnerabilities
- Verify rate limiting works

### Documentation

- Update README.md for new features
- Add inline documentation for complex code
- Update API documentation
- Include examples where helpful

### Performance Considerations

- Profile code changes for performance impact
- Optimize hot paths when necessary
- Consider memory allocation patterns
- Test with realistic data sizes

### Security Considerations

- Never commit sensitive data (tokens, keys, etc.)
- Validate all user inputs
- Use secure coding practices
- Run security tests before submission

## Review Process

1. **Automated checks** must pass (CI/CD pipeline)
2. **Code review** by maintainers
3. **Testing** verification
4. **Documentation** review
5. **Security** assessment for sensitive changes

## Release Process

1. Version numbers follow [Semantic Versioning](https://semver.org/)
2. Releases are created through GitHub releases
3. Changelog is maintained in CHANGELOG.md
4. Docker images are built and pushed automatically

## Getting Help

- **GitHub Issues**: For bugs and feature requests
- **GitHub Discussions**: For questions and general discussion
- **Documentation**: Check the docs/ directory
- **Examples**: Look at test files for usage examples

## Recognition

Contributors are recognized in:
- GitHub contributors page
- Release notes for significant contributions
- Project documentation

Thank you for contributing to live-webrtc-go! 🎉