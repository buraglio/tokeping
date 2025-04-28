const ctx = document.getElementById('latencyChart').getContext('2d');
const data = { labels: [], datasets: [{ label: 'Latency (ms)', data: [], fill: false, borderWidth: 1 }] };
const chart = new Chart(ctx, { type: 'line', data, options: { animation: false, scales: { y: { beginAtZero: true } } } });
const ws = new WebSocket(`ws://${window.location.host}/ws`);
ws.onmessage = e => {
    const m = JSON.parse(e.data);
    const label = new Date(m.Time * 1000).toLocaleTimeString();
    chart.data.labels.push(label);
    chart.data.datasets[0].data.push(m.Latency);
    if (chart.data.labels.length > 50) {
        chart.data.labels.shift();
        chart.data.datasets[0].data.shift();
    }
    chart.update();
};
