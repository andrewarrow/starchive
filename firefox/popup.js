document.getElementById('fetchButton').addEventListener('click', () => {
  chrome.runtime.sendMessage({ type: "fetchData" }, (response) => {
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