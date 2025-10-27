# WebStack CLI - Installation Methods

This document outlines various ways to install and distribute the WebStack CLI tool.

## Quick Install Options

### 1. CLI Tool Only (Recommended for most users)
```bash
curl -fsSL https://your-domain.com/install-cli.sh | sudo bash
```

This method:
- ✅ Downloads and installs the WebStack CLI binary
- ✅ Creates minimal configuration directories
- ✅ No system service integration
- ✅ Perfect for manual web stack management

### 2. CLI Tool + Optional Service Integration
```bash
curl -fsSL https://your-domain.com/install.sh | sudo bash
```

This method:
- ✅ Installs the CLI tool
- ✅ Asks if you want service integration
- ✅ Optional systemd service for automation
- ✅ Optional cron jobs for maintenance

### 3. Service Integration Only (for existing CLI installations)
```bash
curl -fsSL https://your-domain.com/install-service.sh | sudo bash
```

This method:
- ✅ Adds service integration to existing CLI installation
- ✅ Systemd service with automatic configuration reloading
- ✅ Cron jobs for SSL renewal and maintenance
- ✅ Log rotation and monitoring setup

## Distribution Methods

### 1. GitHub Releases (Automated)

**Setup:**
1. Push code to GitHub repository
2. Create a tag: `git tag v1.0.0 && git push origin v1.0.0`
3. GitHub Actions automatically builds and releases binaries

**Installation:**
```bash
# Download latest release
wget https://github.com/yourusername/webstack-cli/releases/latest/download/webstack-linux-amd64

# Make executable and install
chmod +x webstack-linux-amd64
sudo mv webstack-linux-amd64 /usr/local/bin/webstack
```

### 2. APT Repository (Ubuntu/Debian)

**Setup APT Repository:**
```bash
curl -fsSL https://your-domain.com/install-apt.sh | sudo bash
```

**Install via APT:**
```bash
sudo apt update
sudo apt install webstack-cli
```

### 3. Snap Package

**Install from Snap Store:**
```bash
sudo snap install webstack-cli --classic
```

**Local Snap Build:**
```bash
snapcraft
sudo snap install --dangerous webstack-cli_*.snap
```

### 4. Docker Container

**Pull and run:**
```bash
docker pull your-registry/webstack-cli:latest
docker run --rm -it your-registry/webstack-cli:latest --help
```

**Build locally:**
```bash
docker build -t webstack-cli .
docker run --rm -it webstack-cli --help
```

### 5. Manual Binary Download

**Direct download:**
```bash
# Linux AMD64
wget https://github.com/yourusername/webstack-cli/releases/download/v1.0.0/webstack-linux-amd64

# Linux ARM64
wget https://github.com/yourusername/webstack-cli/releases/download/v1.0.0/webstack-linux-arm64

# macOS AMD64
wget https://github.com/yourusername/webstack-cli/releases/download/v1.0.0/webstack-darwin-amd64

# macOS ARM64 (Apple Silicon)
wget https://github.com/yourusername/webstack-cli/releases/download/v1.0.0/webstack-darwin-arm64
```

## Building from Source

### Prerequisites
- Go 1.21 or later
- Git

### Build Steps
```bash
# Clone repository
git clone https://github.com/yourusername/webstack-cli.git
cd webstack-cli

# Build for current platform
go build -o webstack .

# Or use build script for multiple platforms
chmod +x build.sh
./build.sh v1.0.0
```

## Update Methods

### Auto-update (if implemented)
```bash
sudo webstack update
```

### Manual update
```bash
curl -fsSL https://your-domain.com/install.sh | sudo bash
```

### Via package manager
```bash
# APT
sudo apt update && sudo apt upgrade webstack-cli

# Snap
sudo snap refresh webstack-cli
```

## Hosting Options

### 1. GitHub Releases (Free)
- ✅ Automatic builds via GitHub Actions
- ✅ Free hosting for open source
- ✅ Built-in download statistics
- ✅ Easy version management

### 2. Self-hosted Server
```nginx
# Nginx config for hosting binaries
server {
    listen 80;
    server_name your-domain.com;
    
    location /install.sh {
        alias /var/www/webstack/install.sh;
        add_header Content-Type text/plain;
    }
    
    location /releases/ {
        alias /var/www/webstack/releases/;
        autoindex on;
    }
}
```

### 3. CDN Distribution
- CloudFlare R2
- AWS S3 + CloudFront
- DigitalOcean Spaces

### 4. Package Repositories
- **APT**: Host your own repository
- **RPM**: For RedHat/CentOS/Fedora
- **Snap Store**: Ubuntu's universal packages
- **Homebrew**: macOS package manager

## Security Considerations

### GPG Signing
```bash
# Sign releases
gpg --armor --detach-sig webstack-linux-amd64

# Verify downloads
gpg --verify webstack-linux-amd64.sig webstack-linux-amd64
```

### Checksums
Always provide SHA256 checksums:
```bash
sha256sum webstack-* > checksums.txt
```

### HTTPS Only
Ensure all download URLs use HTTPS to prevent man-in-the-middle attacks.

## Analytics and Monitoring

### Download Statistics
- GitHub Releases provides built-in stats
- Use analytics for self-hosted downloads
- Monitor package manager install counts

### Error Reporting
- Implement telemetry (opt-in)
- Log installation errors
- Monitor update success rates

## Recommended Setup

For maximum reach and ease of use:

1. **Primary**: GitHub Releases with automated builds
2. **Backup**: Self-hosted binaries with install script
3. **Convenience**: Package repository (APT/Snap)
4. **Development**: Docker containers for testing

This multi-channel approach ensures users can install via their preferred method while maintaining a reliable primary distribution channel.