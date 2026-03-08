document.addEventListener('DOMContentLoaded', async () => {
    const App = window.go.main.App;
    const logContainer = document.getElementById('log-container');
    const overlay = document.getElementById('loading-overlay');
    const overlayText = overlay?.querySelector('p');
    const runModal = document.getElementById('run-modal');
    const imageSelect = document.getElementById('run-image-select');

    let hostIP = 'localhost';
    window.availableImages = [];
    window.activeContainers = [];

    try {
        hostIP = await App.GetHostIP();
        console.log("Detected Host IP:", hostIP);
    } catch (e) { console.warn("Host IP detection failed."); }

    // --- HELPER FUNCTIONS ---

    const showLoading = (text = "Processing...") => {
        if (overlay) {
            if (overlayText) overlayText.innerText = text;
            overlay.style.display = 'flex';
        }
    };

    const hideLoading = () => { if (overlay) overlay.style.display = 'none'; };

    const addLog = (message, type = 'info') => {
        const time = new Date().toLocaleTimeString();
        const div = document.createElement('div');
        div.className = `log-entry ${type}`;
        div.innerText = `[${time}] ${message}`;
        logContainer.prepend(div);
    };

    const onClick = (id, fn) => {
        const el = document.getElementById(id);
        if (el) el.addEventListener('click', fn);
    };

    // --- RESOURCE VALIDATION ---
    const validateResource = (value, type) => {
        if (!value) return true; // Optional field

        if (type === 'cpu') {
            const cpu = parseFloat(value);
            if (isNaN(cpu) || cpu <= 0) return "CPU must be a positive number (e.g. 0.5)";
            if (cpu > 64) return "CPU limit is too high (max 64)";
        }

        if (type === 'memory') {
            const match = value.match(/^(\d+)([mg])$/i);
            if (!match) return "Format: 512m or 1g";
            const num = parseInt(match[1]);
            const unit = match[2].toLowerCase();
            if (num <= 0) return "Memory must be positive";
            if (unit === 'g' && num > 128) return "Memory limit is too high (max 128g)";
            if (unit === 'm' && num > 128000) return "Memory limit is too high";
        }
        return null;
    };

    const applyError = (elId, error) => {
        const el = document.getElementById(elId);
        if (!el) return;
        if (error) {
            el.classList.add('input-error');
            addLog(`Validation Error: ${error}`, 'error');
        } else {
            el.classList.remove('input-error');
        }
    };

    // --- TAB MANAGEMENT ---
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
            document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
            btn.classList.add('active');
            const target = btn.getAttribute('data-target');
            document.getElementById(target).classList.add('active');
            if (target === 'containers-tab') loadContainers();
            if (target === 'images-tab') loadImages();
        });
    });

    // --- CONTAINER MANAGEMENT ---
    const loadContainers = async () => {
        try {
            const res = await App.ListContainers();
            const tbody = document.querySelector('#containers-table tbody');
            tbody.innerHTML = '';
            window.activeContainers = res.containers || [];
            if (window.activeContainers.length === 0) {
                tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;">No running containers.</td></tr>';
                return;
            }
            window.activeContainers.forEach(c => {
                const tr = document.createElement('tr');
                const portDisplay = c.host_port ? `<a href="#" onclick="openPort('${c.host_port}'); return false;" class="port-link">${c.host_port} → ${c.cont_port}</a>` : '-';
                tr.innerHTML = `
                    <td>${c.pid}</td>
                    <td>${c.image}</td>
                    <td>${c.ip}</td>
                    <td>${portDisplay}</td>
                    <td>
                        <div class="resource-inputs" style="display: flex; gap: 5px;">
                            <input type="text" id="mem-${c.pid}" class="small-input" placeholder="RAM" value="${c.memory || ''}" style="width: 70px;">
                            <input type="text" id="cpu-${c.pid}" class="small-input" placeholder="CPU" value="${c.cpus || ''}" style="width: 60px;">
                        </div>
                    </td>
                    <td><span class="badge ${c.status === 'Running' ? 'running' : 'stopped'}">${c.status}</span></td>
                    <td>
                        <button class="btn-success btn-sm" onclick="updateResources('${c.pid}')">Update</button>
                        <button class="btn-danger btn-sm" onclick="stopContainer('${c.pid}')">Stop</button>
                    </td>
                `;
                tbody.appendChild(tr);
            });
            addLog('Container list updated.', 'info');
        } catch (err) { addLog(`LoadContainers error: ${err}`, 'error'); }
    };

    window.openPort = (port) => {
        const url = `http://${hostIP}:${port}`;
        App.OpenURL(url);
    };

    window.updateResources = async (pid) => {
        const memory = document.getElementById(`mem-${pid}`).value.trim();
        const cpus = document.getElementById(`cpu-${pid}`).value.trim();

        const memErr = validateResource(memory, 'memory');
        const cpuErr = validateResource(cpus, 'cpu');

        if (memErr || cpuErr) {
            alert(memErr || cpuErr);
            return;
        }

        try {
            showLoading("Updating Resources...");
            await App.UpdateContainerResources(pid, { memory, cpus });
            addLog(`Resources updated for PID ${pid}.`, 'success');
            setTimeout(loadContainers, 1000);
        } catch (err) { addLog(`Update error: ${err}`, 'error'); }
        finally { setTimeout(hideLoading, 1000); }
    };

    window.stopContainer = async (id) => {
        if (!confirm(`Stop container #${id}?`)) return;
        try {
            showLoading("Stopping Container...");
            await App.DeleteContainer(id);
            addLog(`Container stopped: ${id}`, 'success');
            setTimeout(loadContainers, 1500);
        } catch (err) { addLog(`Stop error: ${err}`, 'error'); }
        finally { setTimeout(hideLoading, 1500); }
    };

    // --- MODAL LOGIC ---
    onClick('btn-run-modal-open', async () => {
        await loadImages();
        imageSelect.innerHTML = '';
        if (window.availableImages.length === 0) {
            const opt = document.createElement('option');
            opt.innerText = "No images available";
            imageSelect.appendChild(opt);
        } else {
            window.availableImages.forEach(img => {
                const opt = document.createElement('option');
                opt.value = img.name;
                opt.innerText = `${img.name} (${img.size})`;
                imageSelect.appendChild(opt);
            });
        }
        runModal.style.display = 'flex';
    });

    onClick('btn-run-modal-close', () => { runModal.style.display = 'none'; });

    onClick('btn-run-submit', async () => {
        const image = imageSelect.value;
        const ports = document.getElementById('run-port-input').value.trim();
        const memory = document.getElementById('run-mem-input').value.trim();
        const cpus = document.getElementById('run-cpu-input').value.trim();

        if (!image || image === "No images available") return alert("Please select a valid image.");

        // Validations
        const memErr = validateResource(memory, 'memory');
        const cpuErr = validateResource(cpus, 'cpu');
        applyError('run-mem-input', memErr);
        applyError('run-cpu-input', cpuErr);

        if (memErr || cpuErr) return;

        showLoading("Starting Container...");
        try {
            const freshList = await App.ListContainers();
            const active = freshList.containers || [];
            if (ports && ports.includes(':')) {
                const hostPort = ports.split(':')[0];
                if (active.find(c => c.host_port === hostPort)) {
                    hideLoading();
                    return alert(`Port ${hostPort} is already in use.`);
                }
            }

            addLog(`Starting ${image}...`, 'info');
            await App.RunContainer({ image, command: [], ports, memory, cpus });

            setTimeout(() => {
                runModal.style.display = 'none';
                loadContainers();
                hideLoading();
                addLog(`Started: ${image}`, 'success');
                // Clear fields
                document.getElementById('run-port-input').value = '';
                document.getElementById('run-mem-input').value = '';
                document.getElementById('run-cpu-input').value = '';
            }, 2500);
        } catch (err) {
            hideLoading();
            addLog(`Start error: ${err}`, 'error');
        }
    });

    // --- IMAGE MANAGEMENT ---
    const loadImages = async () => {
        try {
            const res = await App.ListImages();
            const tbody = document.querySelector('#images-table tbody');
            if (tbody) tbody.innerHTML = '';
            window.availableImages = res.images || [];
            if (tbody) {
                if (window.availableImages.length === 0) {
                    tbody.innerHTML = '<tr><td colspan="4" style="text-align:center;">No images found.</td></tr>';
                } else {
                    window.availableImages.forEach(img => {
                        const tr = document.createElement('tr');
                        tr.innerHTML = `<td>${img.name}</td><td>${img.size}</td><td>${img.created}</td><td><button class="btn-danger btn-sm" onclick="removeImage('${img.name}')">Delete</button></td>`;
                        tbody.appendChild(tr);
                    });
                }
            }
            addLog('Image list updated.', 'info');
        } catch (err) { addLog(`LoadImages error: ${err}`, 'error'); }
    };

    window.removeImage = async (name) => {
        if (!confirm(`Delete image "${name}"?`)) return;
        try {
            showLoading("Removing Image...");
            await App.RemoveImage(name);
            addLog(`Deleted: ${name}`, 'success');
            setTimeout(loadImages, 1000);
        } catch (err) { addLog(`Delete error: ${err}`, 'error'); }
        finally { setTimeout(hideLoading, 1000); }
    };

    onClick('btn-build', async () => {
        const tag = document.getElementById('build-tag-input').value.trim();
        const context = document.getElementById('build-dir-input').value.trim();
        if (!tag || !context) return alert("Fill all fields");
        if (!(await App.CheckMyBoxFile(context))) return alert("MyBoxFile missing!");

        try {
            showLoading("Building Image...");
            await App.BuildImage({ tag, context });
            addLog(`Building: ${tag}`, 'info');

            let attempts = 0;
            const check = setInterval(async () => {
                attempts++;
                const res = await App.ListImages();
                if ((res.images || []).find(img => img.name === tag || img.name === tag + ".tar") || attempts > 20) {
                    clearInterval(check);
                    loadImages();
                    hideLoading();
                    addLog(`Build process completed/checked for: ${tag}`, 'success');
                }
            }, 2000);
        } catch (err) { hideLoading(); addLog(`Build error: ${err}`, 'error'); }
    });

    onClick('btn-select-dir', async () => {
        try {
            const dir = await App.SelectDirectory();
            if (dir) {
                document.getElementById('build-dir-input').value = dir;
                if (!(await App.CheckMyBoxFile(dir))) alert("Warning: 'MyBoxFile' missing!");
            }
        } catch (e) { addLog(`Directory selection failed: ${e}`, 'error'); }
    });

    onClick('btn-refresh-containers', loadContainers);
    onClick('btn-refresh-images', loadImages);
    onClick('btn-clear-logs', () => {
        logContainer.innerHTML = '';
        addLog('Logs cleared.', 'info');
    });

    addLog('System ready.', 'success');
    loadContainers();
    loadImages();
});