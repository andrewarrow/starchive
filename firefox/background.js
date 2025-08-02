chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
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
  
  if (msg.type === "youtubeVideo") {
    fetch("http://localhost:3009/youtube", {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify({ videoId: msg.videoId })
    })
      .then(res => res.json())
      .then(data => {
        console.log("YouTube video ID sent:", msg.videoId, "Response:", data);
        if (sendResponse) sendResponse(data);
      })
      .catch(err => {
        console.error("Error sending YouTube video ID:", err);
        if (sendResponse) sendResponse({ error: err.message });
      });
    return true;
  }
});
