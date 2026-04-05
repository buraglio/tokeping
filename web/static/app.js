const ctx = document.getElementById('latencyChart').getContext('2d');
const data = { datasets: [] };
const chart = new Chart(ctx, { 
    type: 'line', 
    data, 
    options: { 
        animation: false, 
        scales: { 
            x: { type: 'time', time: { unit: 'minute' } },
            y: { beginAtZero: true } 
        } 
    } 
});

// A generated palette
const colors = ["#e6194b", "#3cb44b", "#ffe119", "#4363d8", "#f58231", "#911eb4", "#46f0f0", "#f032e6", "#bcf60c", "#fabebe", "#008080", "#e6beff", "#9a6324", "#fffac8", "#800000", "#aaffc3", "#808000", "#ffd8b1", "#000075", "#808080", "#ffffff", "#000000"];

const ws = new WebSocket(`ws://${window.location.host}/ws`);
ws.onmessage = e => {
    const m = JSON.parse(e.data);
    let dataset = chart.data.datasets.find(ds => ds.label === m.Probe);
    if (!dataset) {
        const c = colors[chart.data.datasets.length % colors.length];
        dataset = { label: m.Probe, data: [], fill: false, borderColor: c, backgroundColor: c, borderWidth: 1 };
        chart.data.datasets.push(dataset);
    }
    
    // Add point
    dataset.data.push({x: m.Time * 1000, y: m.Latency});
    
    // Keep only last 60 minutes
    const cutoff = Date.now() - 3600000;
    dataset.data = dataset.data.filter(p => p.x > cutoff);
    
    chart.update();
};
