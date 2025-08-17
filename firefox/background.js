console.log('[Starchive] Background script loaded');

// Store for transcript content
let storedTranscripts = {};

browser.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  console.log('[Starchive] Received message:', msg.type, msg);
  
  if (msg.type === "fetchData") {
    fetch("http://localhost:3009/data")
      .then(res => res.json())
      .then(data => sendResponse(data))
      .catch(err => {
        console.error(err);
        sendResponse({ error: err.message });
      });
    return true;
  }
  
  if (msg.type === "requestTxt") {
    console.log(`[Starchive] Requesting txt for video ID: ${msg.videoId}`);
    console.log(`[Starchive] sendResponse function available:`, typeof sendResponse);
    
    fetch(`http://localhost:3009/get-txt?id=${msg.videoId}`)
      .then(res => {
        console.log(`[Starchive] Response status for ${msg.videoId}:`, res.status);
        return res.json();
      })
      .then(data => {
        console.log(`[Starchive] Got response for ${msg.videoId}:`, data);
        
        console.log(`[Starchive] About to send response - hasContent: ${data.hasContent}, videoId: ${msg.videoId}`);
        
        const responseObj = {
          hasContent: data.hasContent,
          content: data.content || null,
          videoId: msg.videoId,
          message: data.message || null
        };
        
        console.log(`[Starchive] Sending response object:`, responseObj);
        sendResponse(responseObj);
        console.log(`[Starchive] Response sent for ${msg.videoId}`);
      })
      .catch(err => {
        console.error(`[Starchive] Error getting txt for ${msg.videoId}:`, err);
        const errorResponse = {
          hasContent: false,
          error: err.message,
          videoId: msg.videoId
        };
        console.log(`[Starchive] Sending error response:`, errorResponse);
        sendResponse(errorResponse);
        console.log(`[Starchive] Error response sent for ${msg.videoId}`);
      });
    return true;
  }
  
  if (msg.type === "storeTranscript") {
    console.log(`[Starchive] Storing transcript for ${msg.videoId}, length: ${msg.content.length}`);
    storedTranscripts[msg.videoId] = {
      content: msg.content,
      timestamp: Date.now()
    };
    console.log(`[Starchive] Stored transcripts count: ${Object.keys(storedTranscripts).length}`);
    return true;
  }
  
  if (msg.type === "getStoredTranscript") {
    console.log('[Starchive] Get stored transcript request from popup');
    
    if (Object.keys(storedTranscripts).length === 0) {
      console.log('[Starchive] No stored transcripts available');
      sendResponse({
        success: false,
        error: 'No transcript available. Hover over a YouTube video with a transcript first.'
      });
      return true;
    }
    
    // Get the most recently stored transcript
    const sortedTranscripts = Object.entries(storedTranscripts)
      .sort((a, b) => b[1].timestamp - a[1].timestamp);
    
    const [videoId, transcriptData] = sortedTranscripts[0];
    
    console.log(`[Starchive] Returning transcript for ${videoId}, length: ${transcriptData.content.length}`);
    
    sendResponse({
      success: true,
      videoId: videoId,
      content: transcriptData.content,
      length: transcriptData.content.length
    });
    
    return true;
  }
  
  if (msg.type === "youtubeVideo") {
    // Collect YouTube cookies then post video + cookies to backend
    try {
      browser.cookies.getAll({ domain: "youtube.com" }, (cookies) => {
        if (browser.runtime.lastError) {
          console.error("Error getting cookies:", browser.runtime.lastError);
        }

        const minimalCookies = (cookies || []).map(c => ({
          name: c.name,
          value: c.value,
          domain: c.domain,
          path: c.path || "/",
          expires: c.expirationDate || 0,
          secure: !!c.secure,
          httpOnly: !!c.httpOnly
        }));

        fetch("http://localhost:3009/youtube", {
          method: "POST",
          headers: {
            "Content-Type": "application/json"
          },
          body: JSON.stringify({ videoId: msg.videoId, cookies: minimalCookies })
        })
          .then(res => res.text())
          .then(text => {
            // Try to parse JSON, but fall back to plain text
            let data;
            try { data = JSON.parse(text); } catch (_) { data = { message: text }; }
            console.log("YouTube video sent:", msg.videoId, "Cookies:", minimalCookies.length, "Response:", data);
            if (sendResponse) sendResponse(data);
          })
          .catch(err => {
            console.error("Error sending YouTube video:", err);
            if (sendResponse) sendResponse({ error: err.message });
          });
      });
    } catch (err) {
      console.error("Cookie collection failed:", err);
      fetch("http://localhost:3009/youtube", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ videoId: msg.videoId })
      }).catch(e => console.error("Fallback request failed:", e));
    }
    return true;
  }
});

// Handle toolbar button clicks
browser.browserAction.onClicked.addListener((tab) => {
  console.log('[Starchive] Toolbar button clicked, checking for stored transcripts');
  
  if (Object.keys(storedTranscripts).length === 0) {
    console.log('[Starchive] No stored transcripts available');
    browser.notifications.create({
      type: 'basic',
      iconUrl: 'icon.png',
      title: 'Starchive',
      message: 'No transcript available. Hover over a YouTube video with a transcript first.'
    });
    return;
  }
  
  // Get the most recently stored transcript
  const sortedTranscripts = Object.entries(storedTranscripts)
    .sort((a, b) => b[1].timestamp - a[1].timestamp);
  
  const [videoId, transcriptData] = sortedTranscripts[0];
  
  console.log(`[Starchive] Copying transcript for ${videoId} to clipboard, length: ${transcriptData.content.length}`);
  
  // Copy to clipboard
  navigator.clipboard.writeText(transcriptData.content).then(() => {
    console.log('[Starchive] Transcript copied to clipboard via toolbar button');
    browser.notifications.create({
      type: 'basic',
      iconUrl: 'icon.png',
      title: 'Starchive',
      message: `📋 Transcript for ${videoId} copied to clipboard! (${transcriptData.content.length} chars)`
    });
  }).catch(err => {
    console.error('[Starchive] Failed to copy via toolbar button:', err);
    browser.notifications.create({
      type: 'basic',
      iconUrl: 'icon.png', 
      title: 'Starchive',
      message: '❌ Failed to copy transcript to clipboard'
    });
  });
});
