// tf-blast Documentation Site Application Script

(function () {
  'use strict';

  // --- Theme Management ---
  const THEME_STORAGE_KEY = 'color-scheme';
  const themeToggleBtn = document.getElementById('theme-toggle');
  const colorSchemeMeta = document.querySelector('meta[name="color-scheme"]');

  function getSystemTheme() {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }

  function applyTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    if (colorSchemeMeta) {
      colorSchemeMeta.content = theme;
    }
  }

  const savedTheme = localStorage.getItem(THEME_STORAGE_KEY);
  if (savedTheme) {
    applyTheme(savedTheme);
  }

  if (themeToggleBtn) {
    themeToggleBtn.addEventListener('click', function () {
      const currentTheme = document.documentElement.getAttribute('data-theme') || getSystemTheme();
      const nextTheme = currentTheme === 'dark' ? 'light' : 'dark';
      applyTheme(nextTheme);
      localStorage.setItem(THEME_STORAGE_KEY, nextTheme);
    });
  }

  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function (e) {
    if (!localStorage.getItem(THEME_STORAGE_KEY)) {
      applyTheme(e.matches ? 'dark' : 'light');
    }
  });

  // --- Copy to Clipboard Utility ---
  document.querySelectorAll('.copy-btn').forEach(function (button) {
    button.addEventListener('click', function () {
      const targetId = button.getAttribute('data-target');
      let textToCopy = '';

      if (targetId) {
        const el = document.getElementById(targetId);
        if (el) textToCopy = el.textContent || '';
      } else if (button.getAttribute('data-copy')) {
        textToCopy = button.getAttribute('data-copy');
      }

      if (textToCopy) {
        navigator.clipboard.writeText(textToCopy.trim()).then(function () {
          const originalHTML = button.innerHTML;
          button.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"></polyline></svg> Copied!';
          button.style.color = 'var(--brand-success)';
          setTimeout(function () {
            button.innerHTML = originalHTML;
            button.style.color = '';
          }, 2000);
        }).catch(function (err) {
          console.error('Clipboard copy failed:', err);
        });
      }
    });
  });

  // --- Install Tabs Switcher ---
  const installTabs = document.querySelectorAll('.install-tab');
  const installCmdEl = document.getElementById('install-cmd-text');
  const installCopyBtn = document.getElementById('install-copy-btn');

  const installCommands = {
    curl: 'curl -sSL https://raw.githubusercontent.com/smford/tf-blast/main/install.sh | bash',
    go: 'go install github.com/smford/tf-blast/cmd/tf-blast@latest',
    docker: 'docker pull ghcr.io/smford/tf-blast:latest',
    action: 'uses: smford/tf-blast@v1'
  };

  installTabs.forEach(function (tab) {
    tab.addEventListener('click', function () {
      installTabs.forEach(function (t) { t.classList.remove('active'); });
      tab.classList.add('active');
      const tool = tab.getAttribute('data-tool');
      if (installCommands[tool] && installCmdEl) {
        installCmdEl.textContent = installCommands[tool];
        if (installCopyBtn) {
          installCopyBtn.setAttribute('data-copy', installCommands[tool]);
        }
      }
    });
  });

  // --- Showcase Tabs Switcher ---
  const showcaseTabs = document.querySelectorAll('.showcase-tab');
  const showcasePanels = document.querySelectorAll('.showcase-panel');

  showcaseTabs.forEach(function (tab) {
    tab.addEventListener('click', function () {
      const targetPanelId = tab.getAttribute('data-showcase');
      showcaseTabs.forEach(function (t) { t.classList.remove('active'); });
      showcasePanels.forEach(function (p) { p.style.display = 'none'; });

      tab.classList.add('active');
      const targetPanel = document.getElementById(targetPanelId);
      if (targetPanel) {
        targetPanel.style.display = 'block';
      }
    });
  });

  // --- Dynamic Version & Release Fetching ---
  const REPO = 'smford/tf-blast';

  function updateDOMWithVersion(vData) {
    const version = vData.version || 'v1.0.0';
    const versionNum = vData.version_number || version.replace(/^v/, '');
    const releaseUrl = vData.release_url || `https://github.com/${REPO}/releases/tag/${version}`;

    // Update text elements
    document.querySelectorAll('[data-version]').forEach(function (el) {
      el.textContent = version;
    });
    document.querySelectorAll('[data-version-num]').forEach(function (el) {
      el.textContent = versionNum;
    });
    document.querySelectorAll('[data-release-url]').forEach(function (el) {
      el.setAttribute('href', releaseUrl);
    });

    if (vData.released_at) {
      const dateObj = new Date(vData.released_at);
      const formattedDate = dateObj.toLocaleDateString(undefined, {
        year: 'numeric',
        month: 'short',
        day: 'numeric'
      });
      document.querySelectorAll('[data-release-date]').forEach(function (el) {
        el.textContent = formattedDate;
      });
    }

    // Update binary download URLs
    const baseDownloadUrl = `https://github.com/${REPO}/releases/download/${version}`;
    const downloads = {
      'linux-amd64': `${baseDownloadUrl}/tf-blast_${versionNum}_linux_amd64.tar.gz`,
      'linux-arm64': `${baseDownloadUrl}/tf-blast_${versionNum}_linux_arm64.tar.gz`,
      'darwin-amd64': `${baseDownloadUrl}/tf-blast_${versionNum}_darwin_amd64.tar.gz`,
      'darwin-arm64': `${baseDownloadUrl}/tf-blast_${versionNum}_darwin_arm64.tar.gz`,
      'windows-amd64': `${baseDownloadUrl}/tf-blast_${versionNum}_windows_amd64.zip`
    };

    Object.keys(downloads).forEach(function (key) {
      const linkEl = document.querySelector(`[data-download="${key}"]`);
      if (linkEl) {
        linkEl.setAttribute('href', downloads[key]);
      }
    });
  }

  // 1. First populate immediately from local static version.json
  fetch('version.json')
    .then(function (res) {
      if (res.ok) return res.json();
      throw new Error('Local version.json unavailable');
    })
    .then(function (data) {
      updateDOMWithVersion(data);
    })
    .catch(function (err) {
      console.log('Using default version placeholder:', err);
    });

  // 2. Then attempt to fetch latest live release tag directly from GitHub API
  fetch(`https://api.github.com/repos/${REPO}/releases/latest`)
    .then(function (res) {
      if (res.ok) return res.json();
      throw new Error(`GitHub API returned status ${res.status}`);
    })
    .then(function (ghData) {
      if (ghData && ghData.tag_name) {
        const liveData = {
          version: ghData.tag_name,
          version_number: ghData.tag_name.replace(/^v/, ''),
          released_at: ghData.published_at || ghData.created_at,
          release_url: ghData.html_url
        };
        updateDOMWithVersion(liveData);
      }
    })
    .catch(function (err) {
      // Graceful silence on rate-limiting or offline
      console.debug('GitHub API release lookup fallback to version.json:', err);
    });

})();
