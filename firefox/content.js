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

checkForYouTubeVideo();

let lastUrl = window.location.href;
const observer = new MutationObserver(() => {
  if (lastUrl !== window.location.href) {
    lastUrl = window.location.href;
    checkForYouTubeVideo();
  }
});

observer.observe(document, { subtree: true, childList: true });

chrome.runtime.sendMessage({ type: "fetchData" });