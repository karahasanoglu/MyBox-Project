document.addEventListener('DOMContentLoaded', () => {
    const App = window.go.main.App; // Wails App instance
    const logContainer = document.getElementById('log-container');

    // --- YARDIMCI FONKSİYONLAR ---
    
    // İşlem loglarını merkezi olarak tutar
    const addLog = (message, type = 'info') => {
        const time = new Date().toLocaleTimeString();
        const div = document.createElement('div');
        div.className = `log-entry ${type}`;
        div.innerText = `[${time}] ${message}`;
        logContainer.prepend(div);
    };

    // DOM tabanlı olay dinleyicisi ekleyici
    const onClick = (id, fn) => {
        const el = document.getElementById(id);
        if (el) el.addEventListener('click', fn);
    };

    // --- SEKME (TAB) YÖNETİMİ ---
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
            document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
            
            btn.classList.add('active');
            document.getElementById(btn.getAttribute('data-target')).classList.add('active');
            
            // Sekme değiştiğinde verileri tazele
            if (btn.getAttribute('data-target') === 'containers-tab') loadContainers();
            if (btn.getAttribute('data-target') === 'images-tab') loadImages();
        });
    });

    // --- KONTEYNER YÖNETİMİ ---
    const loadContainers = async () => {
        try {
            const res = await App.ListContainers();
            const tbody = document.querySelector('#containers-table tbody');
            tbody.innerHTML = '';
            
            if (!res.containers || res.containers.length === 0) {
                tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;">Çalışan konteyner yok.</td></tr>';
                return;
            }

            res.containers.forEach(c => {
                const tr = document.createElement('tr');
                // Not: c.id genellikle PID ile aynıdır
                tr.innerHTML = `
                    <td>${c.pid}</td>
                    <td>${c.image}</td>
                    <td>${c.ip}</td>
                    <td>${c.host_port ? `${c.host_port}->${c.cont_port}` : '-'}</td>
                    <td>
                        <div class="resource-inputs" style="display: flex; gap: 5px;">
                            <input type="text" id="mem-${c.pid}" class="small-input" placeholder="RAM (512m)" value="${c.memory || ''}" style="width: 70px; padding: 4px;">
                            <input type="text" id="cpu-${c.pid}" class="small-input" placeholder="CPU (0.5)" value="${c.cpus || ''}" style="width: 60px; padding: 4px;">
                        </div>
                    </td>
                    <td><span class="badge ${c.status === 'Running' ? 'running' : 'stopped'}">${c.status}</span></td>
                    <td>
                        <button class="btn-success btn-sm" onclick="updateResources('${c.pid}')">Güncelle</button>
                        <button class="btn-danger btn-sm" onclick="stopContainer('${c.pid}')">Durdur</button>
                    </td>
                `;
                tbody.appendChild(tr);
            });
            addLog('Konteyner listesi güncellendi.', 'info');
        } catch (err) { addLog(`Konteynerler alınamadı: ${err}`, 'error'); }
    };

    // Dinamik Kaynak Güncelleme (Yeni Fonksiyon)
    window.updateResources = async (pid) => {
        const memory = document.getElementById(`mem-${pid}`).value.trim();
        const cpus = document.getElementById(`cpu-${pid}`).value.trim();

        try {
            addLog(`Kaynaklar güncelleniyor (PID: ${pid}): ${memory || 'limitsiz'} RAM, ${cpus || 'limitsiz'} CPU`, 'warning');
            // Backend UpdateContainerResourcesHandler'ı tetikler
            await App.UpdateContainerResources(pid, { memory, cpus });
            addLog(`PID ${pid} için kaynaklar başarıyla güncellendi.`, 'success');
        } catch (err) {
            addLog(`Güncelleme hatası (PID: ${pid}): ${err}`, 'error');
        }
    };

    // Konteynere durdurma sinyali gönderir
    window.stopContainer = async (id) => {
        if (!confirm(`${id} PID'li konteyneri durdurmak istediğinize emin misiniz?`)) return;
        try {
            addLog(`Konteyner durduruluyor: ${id}`, 'warning');
            await App.DeleteContainer(id);
            addLog(`Konteyner başarıyla durduruldu: ${id}`, 'success');
            loadContainers();
        } catch (err) { addLog(`Durdurma hatası (${id}): ${err}`, 'error'); }
    };

    onClick('btn-refresh-containers', loadContainers);
    
    onClick('btn-run-modal', () => {
        const group = document.getElementById('run-container-group');
        group.style.display = group.style.display === 'none' ? 'flex' : 'none';
    });

    // Yeni konteyner başlatırken limitleri de gönderir
    onClick('btn-run-submit', async () => {
        const image = document.getElementById('run-image-input').value.trim();
        const ports = document.getElementById('run-port-input').value.trim();
        const memory = document.getElementById('run-mem-input') ? document.getElementById('run-mem-input').value.trim() : "";
        const cpus = document.getElementById('run-cpu-input') ? document.getElementById('run-cpu-input').value.trim() : "";

        if (!image) return addLog('Başlatmak için bir imaj adı girmelisiniz.', 'warning');

        try {
            addLog(`Konteyner başlatılıyor (${image})...`, 'info');
            // Backend RunContainerHandler'ı tetikler
            await App.RunContainer({ image, command: [], ports, memory, cpus }); 
            addLog(`Konteyner başarıyla başlatıldı: ${image}`, 'success');
            
            // Formu temizle
            document.getElementById('run-image-input').value = '';
            document.getElementById('run-port-input').value = '';
            if(document.getElementById('run-mem-input')) document.getElementById('run-mem-input').value = '';
            if(document.getElementById('run-cpu-input')) document.getElementById('run-cpu-input').value = '';
            
            loadContainers();
        } catch (err) { addLog(`Başlatma hatası: ${err}`, 'error'); }
    });

    // --- İMAJ YÖNETİMİ ---
    const loadImages = async () => {
        try {
            const res = await App.ListImages();
            const tbody = document.querySelector('#images-table tbody');
            tbody.innerHTML = '';
            
            if (!res.images || res.images.length === 0) {
                tbody.innerHTML = '<tr><td colspan="4" style="text-align:center;">İmaj bulunamadı.</td></tr>';
                return;
            }

            res.images.forEach(img => {
                const tr = document.createElement('tr');
                tr.innerHTML = `
                    <td>${img.name}</td>
                    <td>${img.size}</td>
                    <td>${img.created}</td>
                    <td><button class="btn-danger btn-sm" onclick="removeImage('${img.name}')">Sil</button></td>
                `;
                tbody.appendChild(tr);
            });
            addLog('İmaj listesi güncellendi.', 'info');
        } catch (err) { addLog(`İmajlar alınamadı: ${err}`, 'error'); }
    };

    window.removeImage = async (name) => {
        if (!confirm(`"${name}" imajını silmek istediğinize emin misiniz?`)) return;
        try {
            addLog(`İmaj siliniyor: ${name}`, 'warning');
            await App.RemoveImage(name);
            addLog(`İmaj başarıyla silindi: ${name}`, 'success');
            loadImages();
        } catch (err) { addLog(`İmaj silme hatası (${name}): ${err}`, 'error'); }
    };

    onClick('btn-refresh-images', loadImages);

    onClick('btn-select-dir', async () => {
        try {
            const dir = await App.SelectDirectory();
            if (dir) {
                document.getElementById('build-dir-input').value = dir;
                addLog(`Build dizini seçildi: ${dir}`, 'info');
            }
        } catch (err) { addLog(`Klasör seçilemedi: ${err}`, 'error'); }
    });

    onClick('btn-build', async () => {
        const tag = document.getElementById('build-tag-input').value.trim();
        const context = document.getElementById('build-dir-input').value.trim();
        
        if (!tag || !context) return addLog('Lütfen imaj adı ve klasör yolu girin.', 'warning');

        try {
            addLog(`İmaj derleniyor: ${tag} (Bu işlem biraz sürebilir...)`, 'info');
            await App.BuildImage({ tag, context });
            addLog(`Build işlemi arka planda başlatıldı: ${tag}`, 'success');
            setTimeout(loadImages, 3000); // 3 sn sonra tabloyu yenilemeyi dene
        } catch (err) { addLog(`Build hatası: ${err}`, 'error'); }
    });

    // --- LOG YÖNETİMİ ---
    onClick('btn-clear-logs', () => {
        logContainer.innerHTML = '';
        addLog('Loglar temizlendi.', 'info');
    });

    // Başlangıç yüklemeleri
    addLog('Arayüz hazır. Sistem başlatıldı.', 'success');
    loadContainers(); // Uygulama açılışında konteynerleri getir
});