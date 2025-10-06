// Theme Toggle Functionality
(function() {
  'use strict';
  
  // Initialize theme from localStorage or default to light
  const currentTheme = localStorage.getItem('theme') || 'light';
  document.documentElement.setAttribute('data-theme', currentTheme);
  
  // Create theme toggle button
  function createThemeToggle() {
    const toggleBtn = document.createElement('button');
    toggleBtn.className = 'theme-toggle';
    toggleBtn.setAttribute('aria-label', 'Toggle theme');
    toggleBtn.innerHTML = '<span class="theme-toggle-icon">🌙</span>';
    
    toggleBtn.addEventListener('click', toggleTheme);
    document.body.appendChild(toggleBtn);
    
    updateToggleIcon(toggleBtn);
  }
  
  // Toggle between light and dark themes
  function toggleTheme() {
    const currentTheme = document.documentElement.getAttribute('data-theme');
    const newTheme = currentTheme === 'light' ? 'dark' : 'light';
    
    document.documentElement.setAttribute('data-theme', newTheme);
    localStorage.setItem('theme', newTheme);
    
    const toggleBtn = document.querySelector('.theme-toggle');
    updateToggleIcon(toggleBtn);
  }
  
  // Update toggle button icon
  function updateToggleIcon(btn) {
    const theme = document.documentElement.getAttribute('data-theme');
    btn.innerHTML = theme === 'light' 
      ? '<span class="theme-toggle-icon">🌙</span>'
      : '<span class="theme-toggle-icon">☀️</span>';
  }
  
  // Initialize on DOM ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', createThemeToggle);
  } else {
    createThemeToggle();
  }
})();
