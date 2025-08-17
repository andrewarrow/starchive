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
          
          // Store transcript content for toolbar button
          if (response.hasContent && response.content) {
            console.log('[Starchive] Storing transcript for', videoId, 'content length:', response.content.length);
            browser.runtime.sendMessage({
              type: "storeTranscript",
              videoId: videoId,
              content: response.content
            });
          } else {
            console.log('[Starchive] No transcript to store - hasContent:', response.hasContent, 'content length:', response.content ? response.content.length : 'null');
          }
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

function copyToClipboard(content, videoId) {
  console.log('[Starchive] Copying transcript to clipboard for', videoId, 'content preview:', content.substring(0, 100) + '...');
  console.log('[Starchive] Content type:', typeof content, 'length:', content.length);
  
  navigator.clipboard.writeText(content).then(() => {
    console.log('[Starchive] Transcript copied to clipboard successfully for', videoId);
    
    // Show a brief notification
    const notification = document.createElement('div');
    notification.textContent = '📋 Transcript copied to clipboard!';
    notification.style.cssText = `
      position: fixed;
      top: 20px;
      right: 20px;
      background: #4CAF50;
      color: white;
      padding: 12px 20px;
      border-radius: 6px;
      font-family: Arial, sans-serif;
      font-size: 14px;
      z-index: 10001;
      opacity: 0;
      transition: opacity 0.3s ease;
      box-shadow: 0 4px 12px rgba(0,0,0,0.2);
    `;
    
    document.body.appendChild(notification);
    
    setTimeout(() => notification.style.opacity = '1', 10);
    
    setTimeout(() => {
      notification.style.opacity = '0';
      setTimeout(() => {
        if (notification.parentNode) {
          notification.parentNode.removeChild(notification);
        }
      }, 300);
    }, 2500);
    
  }).catch(err => {
    console.error('[Starchive] Failed to copy transcript to clipboard:', err);
    
    // Fallback: create a temporary text area
    const textArea = document.createElement('textarea');
    textArea.value = content;
    textArea.style.position = 'fixed';
    textArea.style.opacity = '0';
    document.body.appendChild(textArea);
    textArea.select();
    
    try {
      document.execCommand('copy');
      console.log('[Starchive] Transcript copied using fallback method for', videoId);
    } catch (fallbackErr) {
      console.error('[Starchive] Fallback copy method also failed:', fallbackErr);
    }
    
    document.body.removeChild(textArea);
  });
}

function showCopyButton(element, content, videoId) {
  console.log('[Starchive] Creating copy button for', videoId);
  
  // Find thumbnail for positioning
  let thumbnail = element.querySelector('img, yt-image img, ytd-thumbnail img');
  if (!thumbnail) {
    const parentContainer = element.closest('ytd-video-renderer, ytd-compact-video-renderer, ytd-grid-video-renderer');
    if (parentContainer) {
      thumbnail = parentContainer.querySelector('img, yt-image img, ytd-thumbnail img');
    }
  }
  if (!thumbnail && element.parentNode) {
    thumbnail = element.parentNode.querySelector('img, yt-image img, ytd-thumbnail img');
  }
  
  if (!thumbnail) {
    console.log('[Starchive] No thumbnail found for copy button');
    return;
  }
  
  // Create copy button
  const copyButton = document.createElement('div');
  copyButton.innerHTML = '📋';
  copyButton.style.cssText = `
    position: absolute;
    top: 8px;
    right: 8px;
    width: 32px;
    height: 32px;
    background: rgba(76, 175, 80, 0.9);
    color: white;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 16px;
    cursor: pointer;
    z-index: 99999;
    opacity: 0;
    transition: all 0.3s ease;
    box-shadow: 0 2px 8px rgba(0,0,0,0.3);
    user-select: none;
  `;
  
  // Position button on thumbnail
  const rect = thumbnail.getBoundingClientRect();
  copyButton.style.position = 'absolute';
  copyButton.style.left = (rect.right - 40 + window.scrollX) + 'px';
  copyButton.style.top = (rect.top + 8 + window.scrollY) + 'px';
  
  console.log('[Starchive] Copy button positioned at:', { 
    left: rect.right - 40 + window.scrollX, 
    top: rect.top + 8 + window.scrollY, 
    thumbnailRect: rect 
  });
  
  // Add click handler with proper user activation
  copyButton.addEventListener('click', (e) => {
    e.preventDefault();
    e.stopPropagation();
    console.log('[Starchive] Copy button clicked for', videoId);
    
    navigator.clipboard.writeText(content).then(() => {
      console.log('[Starchive] Transcript copied successfully via button click');
      
      // Show success feedback
      copyButton.innerHTML = '✓';
      copyButton.style.background = 'rgba(46, 125, 50, 0.9)';
      
      showCopyNotification();
      
      setTimeout(() => {
        copyButton.innerHTML = '📋';
        copyButton.style.background = 'rgba(76, 175, 80, 0.9)';
      }, 1000);
      
    }).catch(err => {
      console.error('[Starchive] Button click copy failed:', err);
      copyButton.innerHTML = '❌';
      copyButton.style.background = 'rgba(244, 67, 54, 0.9)';
      
      setTimeout(() => {
        copyButton.innerHTML = '📋';
        copyButton.style.background = 'rgba(76, 175, 80, 0.9)';
      }, 1000);
    });
  });
  
  document.body.appendChild(copyButton);
  console.log('[Starchive] Copy button added to DOM');
  
  // Fade in
  setTimeout(() => {
    copyButton.style.opacity = '1';
    copyButton.style.transform = 'scale(1.1)';
    console.log('[Starchive] Copy button faded in');
    setTimeout(() => copyButton.style.transform = 'scale(1)', 200);
  }, 100);
  
  // Auto-remove after delay
  setTimeout(() => {
    copyButton.style.opacity = '0';
    setTimeout(() => {
      if (copyButton.parentNode) {
        copyButton.parentNode.removeChild(copyButton);
      }
    }, 300);
  }, 3000);
}

function showCopyNotification() {
  const notification = document.createElement('div');
  notification.textContent = '📋 Transcript copied to clipboard!';
  notification.style.cssText = `
    position: fixed;
    top: 20px;
    right: 20px;
    background: #4CAF50;
    color: white;
    padding: 12px 20px;
    border-radius: 6px;
    font-family: Arial, sans-serif;
    font-size: 14px;
    z-index: 10003;
    opacity: 0;
    transition: opacity 0.3s ease;
    box-shadow: 0 4px 12px rgba(0,0,0,0.2);
  `;
  
  document.body.appendChild(notification);
  
  setTimeout(() => notification.style.opacity = '1', 10);
  
  setTimeout(() => {
    notification.style.opacity = '0';
    setTimeout(() => {
      if (notification.parentNode) {
        notification.parentNode.removeChild(notification);
      }
    }, 300);
  }, 2500);
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
  const left = (rect.left + rect.width / 2 - tooltip.offsetWidth / 2 + window.scrollX);
  const top = (rect.top - tooltip.offsetHeight - 8 + window.scrollY);
  
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