// TempFiles Web UI - JavaScript Functionality

class TempFilesUI {
    constructor() {
        this.fileInput = document.getElementById('fileInput');
        this.uploadArea = document.getElementById('uploadArea');
        this.uploadForm = document.getElementById('uploadForm');
        this.progressContainer = document.querySelector('.progress-container');
        this.progressFill = document.querySelector('.progress-fill');
        this.progressText = document.querySelector('.progress-text');
        this.fileInfo = document.querySelector('.file-info');
        this.alertContainer = document.querySelector('.alert');
        
        // Read max file size from data attribute, fallback to 100MB
        const uploadAreaEl = document.getElementById('uploadArea');
        this.maxFileSize = uploadAreaEl?.dataset.maxFileSize 
            ? parseInt(uploadAreaEl.dataset.maxFileSize, 10) 
            : 100 * 1024 * 1024;
        this.maxFileSizeHuman = uploadAreaEl?.dataset.maxFileSizeHuman || '100.0 MB';
        this.selectedFile = null;
        
        this.init();
    }
    
    init() {
        this.setupEventListeners();
        this.setupTheme();
        this.setupDragAndDrop();
    }
    
    setupEventListeners() {
        // File input change
        this.fileInput?.addEventListener('change', (e) => {
            this.handleFileSelect(e.target.files[0]);
        });
        
        // Upload area click
        this.uploadArea?.addEventListener('click', () => {
            this.fileInput?.click();
        });
        
        // Form submit
        this.uploadForm?.addEventListener('submit', (e) => {
            if (!this.selectedFile) {
                e.preventDefault();
                this.showAlert('Please select a file first', 'error');
                return;
            }
            
            // Ensure the file input has the selected file
            if (this.fileInput && this.selectedFile) {
                const dt = new DataTransfer();
                dt.items.add(this.selectedFile);
                this.fileInput.files = dt.files;
            }
            
            // Show progress
            this.showProgress();
            this.disableUploadArea();
            
            // Let form submit normally - don't prevent default
            // The form will submit to / with POST and proper headers
        });
        
        // Theme toggle - cleaned up
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('theme-toggle')) {
                this.toggleTheme();
            }
        });
        
        // Copy button: supports data-text (literal) or data-content-id (copy element text)
        document.addEventListener('click', (e) => {
            const btn = e.target.closest('.copy-btn');
            if (!btn) return;
            let text = btn.dataset.text;
            if (!text && btn.dataset.contentId) {
                const el = document.getElementById(btn.dataset.contentId);
                if (el) {
                    const lines = el.querySelectorAll('.paste-line');
                    if (lines.length > 0) {
                        text = Array.prototype.map.call(lines, (n) => n.textContent).join('\n');
                    } else {
                        text = (el.querySelector('code') || el).innerText;
                    }
                }
            }
            if (text != null) {
                this.copyToClipboard(text, btn);
            }
        });
    }
    
    setupTheme() {
        // Check if user has a saved preference
        const savedTheme = localStorage.getItem('tempfiles-theme');
        
        let theme;
        if (savedTheme) {
            // User has explicitly chosen a theme
            theme = savedTheme;
        } else {
            // No saved preference, detect system preference
            theme = this.getSystemTheme();
        }
        
        document.body.setAttribute('data-theme', theme);
        this.updateThemeIcon();
        
        // Listen for system theme changes
        this.setupSystemThemeListener();
    }
    
    getSystemTheme() {
        // Check if browser supports prefers-color-scheme
        if (window.matchMedia) {
            const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
            return isDark ? 'dark' : 'light';
        }
        // Fallback to dark if not supported
        return 'dark';
    }
    
    setupSystemThemeListener() {
        // Only listen to system changes if user hasn't set a preference
        if (!localStorage.getItem('tempfiles-theme') && window.matchMedia) {
            const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
            
            mediaQuery.addEventListener('change', (e) => {
                // Only apply system changes if user hasn't manually set a theme
                if (!localStorage.getItem('tempfiles-theme')) {
                    const newTheme = e.matches ? 'dark' : 'light';
                    document.body.setAttribute('data-theme', newTheme);
                    this.updateThemeIcon();
                }
            });
        }
    }
    
    toggleTheme() {
        const currentTheme = document.body.getAttribute('data-theme');
        const savedTheme = localStorage.getItem('tempfiles-theme');
        
        let newTheme;
        if (!savedTheme) {
            // Currently using system theme, switch to opposite of current
            newTheme = currentTheme === 'dark' ? 'light' : 'dark';
        } else if (savedTheme === 'dark') {
            newTheme = 'light';
        } else if (savedTheme === 'light') {
            // Go back to system theme
            newTheme = this.getSystemTheme();
            localStorage.removeItem('tempfiles-theme'); // Remove saved preference
        } else {
            newTheme = 'dark';
        }
        
        // Always save the preference unless we're going back to system
        if (savedTheme !== 'light') {
            localStorage.setItem('tempfiles-theme', newTheme);
        }
        
        document.body.setAttribute('data-theme', newTheme);
        this.updateThemeIcon();
        
        // Re-setup system listener if we're back to auto
        if (!localStorage.getItem('tempfiles-theme')) {
            this.setupSystemThemeListener();
        }
    }
    
    updateThemeIcon() {
        const themeToggle = document.querySelector('.theme-toggle');
        if (themeToggle) {
            const currentTheme = document.body.getAttribute('data-theme');
            const savedTheme = localStorage.getItem('tempfiles-theme');
            
            if (!savedTheme) {
                // Using system theme - show auto icon
                themeToggle.innerHTML = '🔄';
                themeToggle.title = `Auto (following system ${currentTheme}) - Click to override`;
            } else {
                // User has set a preference - show theme-specific icon
                themeToggle.innerHTML = currentTheme === 'dark' ? '🌙' : '☀️';
                themeToggle.title = `${currentTheme === 'dark' ? 'Dark' : 'Light'} mode - Click to change`;
            }
        }
    }
    
    setupDragAndDrop() {
        if (!this.uploadArea) return;
        
        // Prevent default drag behaviors
        ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
            this.uploadArea.addEventListener(eventName, this.preventDefaults, false);
            document.body.addEventListener(eventName, this.preventDefaults, false);
        });
        
        // Highlight drop area when item is dragged over it
        ['dragenter', 'dragover'].forEach(eventName => {
            this.uploadArea.addEventListener(eventName, () => {
                this.uploadArea.classList.add('dragover');
            }, false);
        });
        
        ['dragleave', 'drop'].forEach(eventName => {
            this.uploadArea.addEventListener(eventName, () => {
                this.uploadArea.classList.remove('dragover');
            }, false);
        });
        
        // Handle dropped files
        this.uploadArea.addEventListener('drop', (e) => {
            const files = e.dataTransfer.files;
            if (files.length > 0) {
                this.handleFileSelect(files[0]);
            }
        }, false);
    }
    
    preventDefaults(e) {
        e.preventDefault();
        e.stopPropagation();
    }
    
    handleFileSelect(file) {
        if (!file) return;
        
        // Validate file size
        if (file.size > this.maxFileSize) {
            this.showAlert(`File size exceeds ${this.maxFileSizeHuman} limit`, 'error');
            return;
        }
        
        this.selectedFile = file;
        this.showFileInfo(file);
        this.hideAlert();
    }
    
    showFileInfo(file) {
        if (!this.fileInfo) return;
        
        const fileName = this.fileInfo.querySelector('.file-name');
        const fileSize = this.fileInfo.querySelector('.file-size');
        
        if (fileName) fileName.textContent = file.name;
        if (fileSize) fileSize.textContent = this.formatFileSize(file.size);
        
        this.fileInfo.style.display = 'block';
        this.fileInfo.classList.add('fade-in');
    }
    
    showProgress() {
        if (this.progressContainer) {
            this.progressContainer.style.display = 'block';
            // Simulate progress for now (real progress would need server-sent events)
            this.animateProgress();
        }
    }
    
    hideProgress() {
        if (this.progressContainer) {
            this.progressContainer.style.display = 'none';
            this.progressFill.style.width = '0%';
        }
    }
    
    animateProgress() {
        let progress = 0;
        const interval = setInterval(() => {
            progress += Math.random() * 15;
            if (progress >= 100) {
                progress = 100;
                clearInterval(interval);
            }
            this.progressFill.style.width = progress + '%';
            this.progressText.textContent = `Uploading... ${Math.round(progress)}%`;
        }, 100);
    }
    
    disableUploadArea() {
        if (this.uploadArea) {
            this.uploadArea.style.pointerEvents = 'none';
            this.uploadArea.style.opacity = '0.7';
        }
        
        const submitBtn = document.querySelector('button[type="submit"]');
        if (submitBtn) {
            submitBtn.disabled = true;
            submitBtn.innerHTML = '<span class="spinner"></span> Uploading...';
        }
    }
    
    enableUploadArea() {
        if (this.uploadArea) {
            this.uploadArea.style.pointerEvents = 'auto';
            this.uploadArea.style.opacity = '1';
        }
        
        const submitBtn = document.querySelector('button[type="submit"]');
        if (submitBtn) {
            submitBtn.disabled = false;
            submitBtn.innerHTML = '🚀 Upload File';
        }
    }
    
    showAlert(message, type = 'error') {
        if (!this.alertContainer) return;
        
        this.alertContainer.className = `alert alert-${type}`;
        this.alertContainer.textContent = message;
        this.alertContainer.style.display = 'block';
        this.alertContainer.classList.add('fade-in');
    }
    
    hideAlert() {
        if (this.alertContainer) {
            this.alertContainer.style.display = 'none';
        }
    }
    
    resetForm() {
        this.selectedFile = null;
        if (this.fileInput) this.fileInput.value = '';
        if (this.fileInfo) this.fileInfo.style.display = 'none';
    }
    
    formatFileSize(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }
    
    copyToClipboard(text, btn) {
        const target = btn || (typeof event !== 'undefined' ? event.target : null);
        const flash = (ok) => {
            if (!target) return;
            const original = target.textContent;
            target.textContent = ok ? '✅ Copied!' : '❌ Copy failed';
            target.disabled = true;
            setTimeout(() => {
                target.textContent = original;
                target.disabled = false;
            }, 1600);
        };

        const useFallback = () => {
            try {
                const ta = document.createElement('textarea');
                ta.value = text;
                ta.setAttribute('readonly', '');
                ta.style.position = 'fixed';
                ta.style.top = '0';
                ta.style.left = '0';
                ta.style.opacity = '0';
                document.body.appendChild(ta);
                ta.focus();
                ta.select();
                ta.setSelectionRange(0, ta.value.length);
                const ok = document.execCommand('copy');
                document.body.removeChild(ta);
                flash(ok);
            } catch (err) {
                console.error('Copy fallback failed:', err);
                flash(false);
            }
        };

        if (navigator.clipboard && window.isSecureContext) {
            navigator.clipboard.writeText(text)
                .then(() => flash(true))
                .catch((err) => {
                    console.warn('Clipboard API failed, using fallback:', err);
                    useFallback();
                });
        } else {
            useFallback();
        }
    }
}

// Countdown Timer for Success Page
class CountdownTimer {
    constructor(expiryTime) {
        this.expiryTime = new Date(expiryTime);
        this.timerElement = document.querySelector('.countdown-time');
        this.start();
    }
    
    start() {
        this.update();
        setInterval(() => this.update(), 1000);
    }
    
    update() {
        const now = new Date();
        const timeLeft = this.expiryTime - now;
        
        if (timeLeft <= 0) {
            this.timerElement.textContent = 'File has expired';
            return;
        }
        
        const hours = Math.floor(timeLeft / (1000 * 60 * 60));
        const minutes = Math.floor((timeLeft % (1000 * 60 * 60)) / (1000 * 60));
        const seconds = Math.floor((timeLeft % (1000 * 60)) / 1000);
        
        this.timerElement.textContent = `${hours}h ${minutes}m ${seconds}s`;
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new TempFilesUI();
    
    // Initialize countdown if on success page
    const expiryTime = document.querySelector('[data-expiry]');
    if (expiryTime) {
        new CountdownTimer(expiryTime.dataset.expiry);
    }
});
