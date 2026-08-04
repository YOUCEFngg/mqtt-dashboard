const ws = new WebSocket(`ws://${window.location.host}/ws`);
const statusDot = document.getElementById('status-dot');
const statusText = document.getElementById('status-text');
const activeTopicsEl = document.getElementById('active-topics');
const metricsEl = document.getElementById('metrics');
const topicTogglesEl = document.getElementById('topic-toggles');
const statsEl = document.getElementById('stats');

let chart;
const chartData = {
    labels: [],
    datasets: []
};

// Track topics
const topics = new Map(); // topic -> { data: [], visible: true, color: string, stats: {} }
const colorPalette = ['#00d4aa', '#3498db', '#e74c3c', '#f39c12', '#9b59b6', '#1abc9c', '#ff6b6b', '#4ecdc4'];

let colorIndex = 0;
function getNextColor() {
    return colorPalette[colorIndex++ % colorPalette.length];
}

// WebSocket handlers
ws.onopen = () => {
    statusDot.className = 'dot connected';
    statusText.textContent = 'Connected';
};

ws.onclose = () => {
    statusDot.className = 'dot disconnected';
    statusText.textContent = 'Disconnected';
};

ws.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    if (msg.type === 'metric') {
        handleMetric(msg);
    }
};

function handleMetric(msg) {
    const topic = msg.topic;
    const topicName = getTopicName(topic);
    
    // Initialize topic if new
    if (!topics.has(topic)) {
        const color = getNextColor();
        topics.set(topic, {
            data: [],
            visible: true,
            color: color,
            stats: { min: Infinity, max: -Infinity, sum: 0, count: 0 }
        });
        
        createTopicToggle(topic, topicName, color);
        createMetricCard(topic, topicName, color);
        createChartDataset(topic, topicName, color);
        createStatsCard(topic, topicName);
    }
    
    const topicInfo = topics.get(topic);
    
    // Update data
    const point = {
        value: msg.value,
        timestamp: new Date(msg.timestamp)
    };
    topicInfo.data.push(point);
    if (topicInfo.data.length > 50) {
        topicInfo.data.shift();
    }
    
    // Update stats
    const stats = topicInfo.stats;
    stats.min = Math.min(stats.min, msg.value);
    stats.max = Math.max(stats.max, msg.value);
    stats.sum += msg.value;
    stats.count++;
    
    // Update UI
    updateMetricCard(topic, msg);
    updateStatsCard(topic);
    
    if (topicInfo.visible) {
        updateChart(topic, msg);
    }
    
    activeTopicsEl.textContent = topics.size;
}

function getTopicName(topic) {
    const parts = topic.split('/');
    return parts[parts.length - 1];
}

// Create topic toggle
function createTopicToggle(topic, name, color) {
    const toggle = document.createElement('div');
    toggle.className = 'toggle active';
    toggle.dataset.topic = topic;
    toggle.innerHTML = `
        <div class="toggle-checkbox"></div>
        <span class="toggle-label">${name}</span>
        <div class="toggle-color" style="background: ${color}"></div>
    `;
    
    toggle.addEventListener('click', () => {
        const info = topics.get(topic);
        info.visible = !info.visible;
        toggle.classList.toggle('active');
        
        // Toggle dataset visibility
        const dataset = chartData.datasets.find(d => d.label === name);
        if (dataset) {
            dataset.hidden = !info.visible;
            chart.update('none');
        }
    });
    
    topicTogglesEl.appendChild(toggle);
}

// Create metric card
function createMetricCard(topic, name, color) {
    const card = document.createElement('div');
    card.className = 'metric-card';
    card.id = `card-${topic.replace(/\//g, '-')}`;
    card.innerHTML = `
        <div class="card-header">
            <span class="topic-name">${name}</span>
            <span class="live-indicator"></span>
        </div>
        <div class="value" style="color: ${color}">--</div>
        <div class="unit">--</div>
        <div class="topic-path">${topic}</div>
        <div class="last-update">Waiting for data...</div>
    `;
    metricsEl.appendChild(card);
}

function updateMetricCard(topic, msg) {
    const card = document.getElementById(`card-${topic.replace(/\//g, '-')}`);
    if (!card) return;
    
    card.querySelector('.value').textContent = msg.value.toFixed(2);
    card.querySelector('.unit').textContent = msg.unit;
    card.querySelector('.last-update').textContent = 
        'Updated: ' + new Date(msg.timestamp).toLocaleTimeString();
}

// Create chart dataset
function createChartDataset(topic, name, color) {
    chartData.datasets.push({
        label: name,
        data: [],
        borderColor: color,
        backgroundColor: color + '20',
        borderWidth: 2,
        tension: 0.4,
        fill: false,
        pointRadius: 3,
        pointHoverRadius: 6,
        hidden: false
    });
}

function updateChart(topic, msg) {
    const time = new Date(msg.timestamp).toLocaleTimeString();
    
    // Add time label if new
    if (!chartData.labels.includes(time)) {
        chartData.labels.push(time);
        if (chartData.labels.length > 30) {
            chartData.labels.shift();
        }
    }
    
    const name = getTopicName(topic);
    const dataset = chartData.datasets.find(d => d.label === name);
    if (dataset) {
        dataset.data.push(msg.value);
        if (dataset.data.length > 30) {
            dataset.data.shift();
        }
    }
    
    chart.update('none');
}

// Create stats card
function createStatsCard(topic, name) {
    const card = document.createElement('div');
    card.className = 'stat-card';
    card.id = `stats-${topic.replace(/\//g, '-')}`;
    card.innerHTML = `
        <div class="stat-label">${name} - Min</div>
        <div class="stat-value" id="min-${topic.replace(/\//g, '-')}">--</div>
    `;
    statsEl.appendChild(card);
    
    const cardMax = document.createElement('div');
    cardMax.className = 'stat-card';
    cardMax.innerHTML = `
        <div class="stat-label">${name} - Max</div>
        <div class="stat-value" id="max-${topic.replace(/\//g, '-')}">--</div>
    `;
    statsEl.appendChild(cardMax);
    
    const cardAvg = document.createElement('div');
    cardAvg.className = 'stat-card';
    cardAvg.innerHTML = `
        <div class="stat-label">${name} - Avg</div>
        <div class="stat-value" id="avg-${topic.replace(/\//g, '-')}">--</div>
    `;
    statsEl.appendChild(cardAvg);
}

function updateStatsCard(topic) {
    const info = topics.get(topic);
    const stats = info.stats;
    const suffix = topic.replace(/\//g, '-');
    
    const minEl = document.getElementById(`min-${suffix}`);
    const maxEl = document.getElementById(`max-${suffix}`);
    const avgEl = document.getElementById(`avg-${suffix}`);
    
    if (minEl) minEl.textContent = stats.min.toFixed(2);
    if (maxEl) maxEl.textContent = stats.max.toFixed(2);
    if (avgEl) avgEl.textContent = (stats.sum / stats.count).toFixed(2);
}

// Chart controls
function clearChart() {
    chartData.labels = [];
    chartData.datasets.forEach(d => d.data = []);
    chart.update();
}

let gridVisible = true;
function toggleGrid() {
    gridVisible = !gridVisible;
    chart.options.scales.x.grid.display = gridVisible;
    chart.options.scales.y.grid.display = gridVisible;
    chart.update();
}

// Initialize chart
chart = new Chart(document.getElementById('mainChart'), {
    type: 'line',
    data: chartData,
    options: {
        responsive: true,
        maintainAspectRatio: false,
        animation: false,
        interaction: {
            intersect: false,
            mode: 'index'
        },
        plugins: {
            legend: {
                labels: {
                    color: '#8b92b4',
                    usePointStyle: true,
                    pointStyle: 'circle',
                    padding: 20
                }
            },
            tooltip: {
                backgroundColor: '#1a1f3d',
                titleColor: '#fff',
                bodyColor: '#8b92b4',
                borderColor: '#2a3060',
                borderWidth: 1,
                padding: 12,
                displayColors: true,
                callbacks: {
                    label: function(context) {
                        return context.dataset.label + ': ' + context.parsed.y.toFixed(2);
                    }
                }
            }
        },
        scales: {
            x: {
                ticks: {
                    color: '#8b92b4',
                    maxTicksLimit: 8,
                    maxRotation: 0
                },
                grid: {
                    color: '#2a3060',
                    display: true
                }
            },
            y: {
                ticks: {
                    color: '#8b92b4'
                },
                grid: {
                    color: '#2a3060',
                    display: true
                }
            }
        }
    }
});