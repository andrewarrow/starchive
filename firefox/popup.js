document.getElementById('fetchButton').addEventListener('click', () => {
  browser.runtime.sendMessage({ type: "fetchData" }, (response) => {
    const resultDiv = document.getElementById('result');
    if (response && response.error) {
      resultDiv.textContent = `Error: ${response.error}`;
    } else if (response) {
      resultDiv.textContent = JSON.stringify(response, null, 2);
    } else {
      resultDiv.textContent = 'No response received';
    }
  });
});

document.getElementById('copyTranscriptButton').addEventListener('click', () => {
  console.log('[Starchive] Copy transcript button clicked in popup');
  browser.runtime.sendMessage({ type: "copyStoredTranscript" }, (response) => {
    const resultDiv = document.getElementById('result');
    if (response && response.success) {
      resultDiv.textContent = `✅ Copied ${response.videoId} (${response.length} chars)`;
      resultDiv.style.background = '#d4edda';
      resultDiv.style.color = '#155724';
    } else if (response && response.error) {
      resultDiv.textContent = `❌ ${response.error}`;
      resultDiv.style.background = '#f8d7da';
      resultDiv.style.color = '#721c24';
    } else {
      resultDiv.textContent = '❌ No response received';
      resultDiv.style.background = '#f8d7da';
      resultDiv.style.color = '#721c24';
    }
  });
});