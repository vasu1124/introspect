// Tutorial Mode with Guided Tours
(function() {
  'use strict';
  
  const tourSteps = [
    {
      element: 'body',
      title: 'Welcome to Introspect!',
      content: 'This guided tour will show you the key features of the application.',
      position: 'center'
    },
    {
      element: '.theme-toggle',
      title: 'Theme Toggle',
      content: 'Click here to switch between light and dark themes for comfortable viewing.',
      position: 'bottom'
    },
    {
      element: 'nav',
      title: 'Navigation',
      content: 'Use the navigation menu to explore different sections of the application.',
      position: 'bottom'
    }
  ];
  
  let currentStep = 0;
  let tourOverlay = null;
  let tourTooltip = null;
  
  // Initialize tutorial mode
  function initTutorialMode() {
    // Check if user has completed tutorial
    if (localStorage.getItem('tutorialCompleted')) {
      return;
    }
    
    // Show tutorial prompt
    const showTutorial = confirm('Would you like a guided tour of the application?');
    if (showTutorial) {
      startTour();
    } else {
      localStorage.setItem('tutorialCompleted', 'true');
    }
  }
  
  // Start guided tour
  function startTour() {
    createOverlay();
    showStep(0);
  }
  
  // Create overlay and tooltip elements
  function createOverlay() {
    tourOverlay = document.createElement('div');
    tourOverlay.className = 'tutorial-overlay';
    tourOverlay.style.cssText = 'position:fixed;top:0;left:0;width:100%;height:100%;background:rgba(0,0,0,0.7);z-index:9999;';
    
    tourTooltip = document.createElement('div');
    tourTooltip.className = 'tutorial-tooltip';
    tourTooltip.style.cssText = 'position:absolute;background:white;padding:20px;border-radius:8px;max-width:300px;z-index:10000;box-shadow:0 4px 6px rgba(0,0,0,0.3);';
    
    document.body.appendChild(tourOverlay);
    document.body.appendChild(tourTooltip);
  }
  
  // Show specific tour step
  function showStep(stepIndex) {
    if (stepIndex >= tourSteps.length) {
      endTour();
      return;
    }
    
    currentStep = stepIndex;
    const step = tourSteps[stepIndex];
    
    // Update tooltip content
    tourTooltip.innerHTML = `
      <h3 style="margin-top:0;">${step.title}</h3>
      <p>${step.content}</p>
      <div style="display:flex;justify-content:space-between;margin-top:15px;">
        <button onclick="window.skipTutorial()" style="padding:8px 16px;border:1px solid #ccc;background:#f5f5f5;border-radius:4px;cursor:pointer;">Skip</button>
        <button onclick="window.nextTutorialStep()" style="padding:8px 16px;border:none;background:#0066cc;color:white;border-radius:4px;cursor:pointer;">Next (${stepIndex + 1}/${tourSteps.length})</button>
      </div>
    `;
    
    // Position tooltip
    positionTooltip(step);
  }
  
  // Position tooltip relative to element
  function positionTooltip(step) {
    const element = document.querySelector(step.element);
    if (element) {
      const rect = element.getBoundingClientRect();
      tourTooltip.style.left = rect.left + 'px';
      tourTooltip.style.top = (rect.bottom + 10) + 'px';
    } else {
      // Center if element not found
      tourTooltip.style.left = '50%';
      tourTooltip.style.top = '50%';
      tourTooltip.style.transform = 'translate(-50%, -50%)';
    }
  }
  
  // Navigate to next step
  window.nextTutorialStep = function() {
    showStep(currentStep + 1);
  };
  
  // Skip tutorial
  window.skipTutorial = function() {
    endTour();
  };
  
  // End tour and cleanup
  function endTour() {
    if (tourOverlay) tourOverlay.remove();
    if (tourTooltip) tourTooltip.remove();
    localStorage.setItem('tutorialCompleted', 'true');
  }
  
  // Initialize on page load
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initTutorialMode);
  } else {
    initTutorialMode();
  }
})();
