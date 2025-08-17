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
  
  // Try multiple ways to find the thumbnail
  let thumbnail = element.querySelector('img, yt-image img, ytd-thumbnail img');
  
  // If not found in the current element, look in parent containers
  if (!thumbnail) {
    const parentContainer = element.closest('ytd-video-renderer, ytd-compact-video-renderer, ytd-grid-video-renderer');
    if (parentContainer) {
      thumbnail = parentContainer.querySelector('img, yt-image img, ytd-thumbnail img');
    }
  }
  
  // If still not found, try looking for thumbnail in siblings
  if (!thumbnail && element.parentNode) {
    thumbnail = element.parentNode.querySelector('img, yt-image img, ytd-thumbnail img');
  }
  
  if (!thumbnail) {
    console.log('[Starchive] No thumbnail found for', videoId, 'element:', element);
    console.log('[Starchive] Element HTML:', element.outerHTML.substring(0, 200));
    // Apply feedback to the element itself if no thumbnail found
    thumbnail = element;
  }

  console.log('[Starchive] Found thumbnail:', thumbnail, 'for video:', videoId);

  const color = hasContent ? '#00ff00' : '#ff0000';
  const message = hasContent ? 'Transcript available' : 'Transcript downloading';
  
  console.log(`[Starchive] Creating ${hasContent ? 'GREEN' : 'RED'} overlay for ${videoId}`);
  
  // Create overlay element
  const overlay = document.createElement('div');
  overlay.style.cssText = `
    position: absolute;
    border: 4px solid ${color};
    box-shadow: 0 0 15px ${color}, inset 0 0 15px ${color};
    pointer-events: none;
    z-index: 10000;
    opacity: 0;
    transition: opacity 0.3s ease;
    background: ${color}22;
  `;
  
  // Position overlay on thumbnail using page coordinates
  const rect = thumbnail.getBoundingClientRect();
  overlay.style.left = (rect.left + window.scrollX) + 'px';
  overlay.style.top = (rect.top + window.scrollY) + 'px';
  overlay.style.width = rect.width + 'px';
  overlay.style.height = rect.height + 'px';
  
  document.body.appendChild(overlay);
  console.log('[Starchive] Overlay created at position:', { left: rect.left, top: rect.top, width: rect.width, height: rect.height });
  
  // Fade in
  setTimeout(() => {
    overlay.style.opacity = '0.8';
    console.log('[Starchive] Overlay faded in');
  }, 10);
  
  setTimeout(() => {
    console.log('[Starchive] Removing overlay for', videoId);
    overlay.style.opacity = '0';
    setTimeout(() => {
      if (overlay.parentNode) {
        overlay.parentNode.removeChild(overlay);
      }
    }, 300);
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