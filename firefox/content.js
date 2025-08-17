function checkForYouTubeVideo() {
  if (window.location.hostname === 'www.youtube.com' || window.location.hostname === 'youtube.com') {
    const urlParams = new URLSearchParams(window.location.search);
    const videoId = urlParams.get('v');
    
    if (videoId && window.location.pathname === '/watch') {
      chrome.runtime.sendMessage({ 
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
      chrome.runtime.sendMessage({
        type: "requestTxt",
        videoId: videoId
      });
    } else {
      console.log('[Starchive] No video ID found in href:', href);
    }
  }
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

chrome.runtime.sendMessage({ type: "fetchData" });