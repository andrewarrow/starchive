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
    document.addEventListener('mouseover', handleMouseOver);
  }
}

function handleMouseOver(event) {
  const target = event.target.closest('a[href*="/watch?v="]');
  if (target) {
    const href = target.getAttribute('href');
    const match = href.match(/[?&]v=([^&]+)/);
    if (match) {
      const videoId = match[1];
      chrome.runtime.sendMessage({
        type: "requestTxt",
        videoId: videoId
      });
    }
  }
}

checkForYouTubeVideo();
setupHoverDetection();

let lastUrl = window.location.href;
const observer = new MutationObserver(() => {
  if (lastUrl !== window.location.href) {
    lastUrl = window.location.href;
    checkForYouTubeVideo();
    setupHoverDetection();
  }
});

observer.observe(document, { subtree: true, childList: true });

chrome.runtime.sendMessage({ type: "fetchData" });