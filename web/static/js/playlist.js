(function () {
    var urlInput = document.getElementById("url");
    var form = document.getElementById("archive-form");
    var smartBtn = document.getElementById("smart-btn");
    var btnLabel = document.getElementById("smart-btn-label");
    var picker = document.getElementById("playlist-picker");
    var pickerTitle = document.getElementById("playlist-picker-title");
    var listEl = document.getElementById("playlist-list");
    var loadingEl = document.getElementById("playlist-loading");
    var errorEl = document.getElementById("playlist-error");
    var archiveBtn = document.getElementById("playlist-archive-btn");
    var selectAllBtn = document.getElementById("playlist-select-all");
    var deselectAllBtn = document.getElementById("playlist-deselect-all");

    if (!smartBtn || !urlInput) return;

    var entries = [];
    var MODE_VIDEO = "video";
    var MODE_LIST = "list";
    var currentMode = MODE_VIDEO;

    function isSingleVideoURL(url) {
        if (!url) return true;
        // youtube.com/watch?v=... or youtu.be/... without &list= → single video
        if (/(?:youtube\.com\/watch\?|youtu\.be\/)/.test(url) && !/[?&]list=/.test(url)) return true;
        // youtube.com/shorts/...
        if (/youtube\.com\/shorts\//.test(url)) return true;
        // Other non-YouTube URLs → default to single video
        if (!/youtube\.com|youtu\.be/.test(url)) return true;
        return false;
    }

    function updateButton() {
        var url = urlInput.value.trim();
        if (isSingleVideoURL(url)) {
            currentMode = MODE_VIDEO;
            btnLabel.textContent = "Archive";
            smartBtn.classList.remove("form-submit-list");
        } else {
            currentMode = MODE_LIST;
            btnLabel.textContent = "List videos";
            smartBtn.classList.add("form-submit-list");
        }
    }

    urlInput.addEventListener("input", updateButton);
    urlInput.addEventListener("change", updateButton);
    updateButton();

    form.addEventListener("submit", function (e) {
        if (currentMode === MODE_LIST) {
            e.preventDefault();
            fetchPlaylist();
        }
    });

    function fetchPlaylist() {
        var url = urlInput.value.trim();
        if (!url) return;

        picker.style.display = "";
        listEl.innerHTML = "";
        loadingEl.style.display = "flex";
        errorEl.style.display = "none";
        entries = [];
        smartBtn.disabled = true;
        btnLabel.textContent = "Loading…";

        fetch("/api/playlist?url=" + encodeURIComponent(url))
            .then(function (r) {
                if (!r.ok) return r.json().then(function (d) { throw new Error(d.error || "Failed"); });
                return r.json();
            })
            .then(function (data) {
                loadingEl.style.display = "none";
                smartBtn.disabled = false;
                updateButton();
                if (!data || data.length === 0) {
                    errorEl.textContent = "No videos found. Is this a playlist or channel URL?";
                    errorEl.style.display = "";
                    return;
                }
                entries = data;
                pickerTitle.textContent = data.length + " video" + (data.length !== 1 ? "s" : "");
                renderEntries();
            })
            .catch(function (err) {
                loadingEl.style.display = "none";
                smartBtn.disabled = false;
                updateButton();
                errorEl.textContent = err.message || "Failed to fetch playlist";
                errorEl.style.display = "";
            });
    }

    function fmtDuration(sec) {
        if (!sec || sec <= 0) return "";
        var h = Math.floor(sec / 3600);
        var m = Math.floor((sec % 3600) / 60);
        var s = Math.floor(sec % 60);
        if (h > 0) return h + ":" + String(m).padStart(2, "0") + ":" + String(s).padStart(2, "0");
        return m + ":" + String(s).padStart(2, "0");
    }

    function escapeHTML(str) {
        var d = document.createElement("div");
        d.textContent = str;
        return d.innerHTML;
    }

    function renderEntries() {
        listEl.innerHTML = entries.map(function (e, i) {
            var dur = fmtDuration(e.duration);
            return '<label class="playlist-entry" for="pl-' + i + '">' +
                '<div class="playlist-entry-thumb">' +
                '<img src="' + escapeHTML(e.thumbnail) + '" alt="" loading="lazy">' +
                (dur ? '<span class="badge-duration">' + dur + '</span>' : '') +
                '</div>' +
                '<div class="playlist-entry-info">' +
                '<span class="playlist-entry-title">' + escapeHTML(e.title || e.id) + '</span>' +
                '<span class="playlist-entry-id">' + escapeHTML(e.id) + '</span>' +
                '</div>' +
                '<input type="checkbox" class="playlist-cb" id="pl-' + i + '" data-idx="' + i + '" checked>' +
                '</label>';
        }).join("");
        updateArchiveBtnCount();
    }

    function updateArchiveBtnCount() {
        var count = listEl.querySelectorAll(".playlist-cb:checked").length;
        archiveBtn.textContent = "Archive selected (" + count + ")";
    }

    listEl.addEventListener("change", updateArchiveBtnCount);

    selectAllBtn.addEventListener("click", function () {
        listEl.querySelectorAll(".playlist-cb").forEach(function (cb) { cb.checked = true; });
        updateArchiveBtnCount();
    });

    deselectAllBtn.addEventListener("click", function () {
        listEl.querySelectorAll(".playlist-cb").forEach(function (cb) { cb.checked = false; });
        updateArchiveBtnCount();
    });

    archiveBtn.addEventListener("click", function () {
        var urls = [];
        listEl.querySelectorAll(".playlist-cb:checked").forEach(function (cb) {
            var idx = parseInt(cb.dataset.idx, 10);
            if (entries[idx]) urls.push(entries[idx].url);
        });

        if (urls.length === 0) return;

        archiveBtn.disabled = true;
        archiveBtn.textContent = "Adding…";

        fetch("/archive/batch", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ urls: urls })
        })
            .then(function () {
                picker.style.display = "none";
                entries = [];
                archiveBtn.disabled = false;
                archiveBtn.textContent = "Archive selected";
            })
            .catch(function () {
                archiveBtn.disabled = false;
                archiveBtn.textContent = "Archive selected";
            });
    });
})();
