function checkForYouTubeVideo() {
  if (window.location.hostname === 'www.youtube.com' || window.location.hostname === 'youtube.com') {
    const urlParams = new URLSearchParams(window.location.search);
    const videoId = urlParams.get('v');
    
    if (videoId && window.location.pathname === '/watch') {
      browser.runtime.sendMessage({ 
        type: "youtubeVideo", 
        videoId: videoId 
      });
    }
  }
}

function setupHoverDetection() {
  if (window.location.hostname === 'www.youtube.com' || window.location.hostname === 'youtube.com') {
    console.log('[Starchive] Setting up hover detection on YouTube');
    document.addEventListener('mouseover', handleMouseOver);
  } else {
    console.log('[Starchive] Not on YouTube, skipping hover detection');
  }
}

function handleMouseOver(event) {
  const target = event.target.closest('a[href*="/watch?v="]');
  if (target) {
    const href = target.getAttribute('href');
    console.log('[Starchive] Hovered over video link:', href);
    const match = href.match(/[?&]v=([^&]+)/);
    if (match) {
      const videoId = match[1];
      console.log('[Starchive] Extracted video ID:', videoId);
      browser.runtime.sendMessage({
        type: "requestTxt",
        videoId: videoId
      }).then((response) => {
        console.log('[Starchive] Got response for', videoId, ':', response);
        if (response) {
          console.log('[Starchive] Response hasContent:', response.hasContent, 'for video:', videoId);
          showVisualFeedback(target, response.hasContent, videoId);
        } else {
          console.log('[Starchive] No response received for', videoId);
        }
      }).catch((error) => {
        console.log('[Starchive] Error sending message for', videoId, ':', error);
      });
    } else {
      console.log('[Starchive] No video ID found in href:', href);
    }
  }
}

function showVisualFeedback(element, hasContent, videoId) {
  console.log('[Starchive] showVisualFeedback called with:', { element, hasContent, videoId });
  
  const thumbnail = element.querySelector('img, yt-image img, ytd-thumbnail img');
  if (!thumbnail) {
    console.log('[Starchive] No thumbnail found for', videoId, 'element:', element);
    console.log('[Starchive] Element HTML:', element.outerHTML.substring(0, 200));
    return;
  }

  console.log('[Starchive] Found thumbnail:', thumbnail, 'for video:', videoId);

  const color = hasContent ? '#00ff00' : '#ff0000';
  const message = hasContent ? 'Transcript available' : 'Transcript downloading';
  
  console.log(`[Starchive] Applying ${hasContent ? 'GREEN' : 'RED'} visual feedback for ${videoId}`);
  console.log('[Starchive] Color:', color, 'Message:', message);
  
  const originalBorder = thumbnail.style.border;
  const originalBoxShadow = thumbnail.style.boxShadow;
  
  console.log('[Starchive] Original styles - border:', originalBorder, 'boxShadow:', originalBoxShadow);
  
  thumbnail.style.border = `3px solid ${color}`;
  thumbnail.style.boxShadow = `0 0 10px ${color}`;
  thumbnail.style.transition = 'all 0.3s ease';
  
  console.log('[Starchive] Applied new styles - border:', thumbnail.style.border, 'boxShadow:', thumbnail.style.boxShadow);
  
  setTimeout(() => {
    console.log('[Starchive] Removing visual feedback for', videoId);
    thumbnail.style.border = originalBorder;
    thumbnail.style.boxShadow = originalBoxShadow;
    console.log('[Starchive] Restored original styles for', videoId);
  }, 1500);
  
  showTooltip(element, message, hasContent);
}

function showTooltip(element, message, isSuccess) {
  console.log('[Starchive] Creating tooltip:', message, 'isSuccess:', isSuccess);
  
  const tooltip = document.createElement('div');
  tooltip.textContent = message;
  tooltip.style.cssText = `
    position: absolute;
    background: ${isSuccess ? '#4CAF50' : '#FF5722'};
    color: white;
    padding: 6px 12px;
    border-radius: 4px;
    font-size: 12px;
    font-family: Arial, sans-serif;
    z-index: 10000;
    pointer-events: none;
    white-space: nowrap;
    box-shadow: 0 2px 8px rgba(0,0,0,0.2);
    opacity: 0;
    transition: opacity 0.3s ease;
  `;
  
  document.body.appendChild(tooltip);
  console.log('[Starchive] Tooltip added to DOM');
  
  const rect = element.getBoundingClientRect();
  const left = (rect.left + rect.width / 2 - tooltip.offsetWidth / 2);
  const top = (rect.top - tooltip.offsetHeight - 8);
  
  tooltip.style.left = left + 'px';
  tooltip.style.top = top + 'px';
  
  console.log('[Starchive] Tooltip positioned at:', { left, top, rect });
  
  setTimeout(() => {
    tooltip.style.opacity = '1';
    console.log('[Starchive] Tooltip faded in');
  }, 10);
  
  setTimeout(() => {
    console.log('[Starchive] Starting tooltip fade out');
    tooltip.style.opacity = '0';
    setTimeout(() => {
      if (tooltip.parentNode) {
        tooltip.parentNode.removeChild(tooltip);
        console.log('[Starchive] Tooltip removed from DOM');
      }
    }, 300);
  }, 2000);
}

console.log('[Starchive] Content script loaded on:', window.location.href);
checkForYouTubeVideo();
setupHoverDetection();

let lastUrl = window.location.href;
const observer = new MutationObserver(() => {
  if (lastUrl !== window.location.href) {
    console.log('[Starchive] URL changed from', lastUrl, 'to', window.location.href);
    lastUrl = window.location.href;
    checkForYouTubeVideo();
    setupHoverDetection();
  }
});

observer.observe(document, { subtree: true, childList: true });

browser.runtime.sendMessage({ type: "fetchData" });