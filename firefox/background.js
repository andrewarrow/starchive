chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (msg.type === "fetchData") {
    fetch("http://localhost:3000/data")
      .then(res => res.json())
      .then(data => sendResponse(data))
      .catch(err => {
        console.error(err);
        sendResponse({ error: err.message });
      });
    return true;
  }
});