(function () {
    const list = document.getElementById("queue-list");
    const clearForm = document.getElementById("clear-form");
    if (!list) return;

    function statusLabel(s) {
        const labels = { pending: "Pending", processing: "Downloading…", done: "Done", error: "Error" };
        return labels[s] || s;
    }

    function escapeHTML(str) {
        const d = document.createElement("div");
        d.textContent = str;
        return d.innerHTML;
    }

    function render(jobs) {
        if (!jobs || jobs.length === 0) {
            list.innerHTML = '<p class="queue-empty">No jobs in the queue.</p>';
            clearForm.style.display = "none";
            return;
        }

        const hasFinished = jobs.some(j => j.status === "done" || j.status === "error");
        clearForm.style.display = hasFinished ? "" : "none";

        list.innerHTML = jobs.map(j => {
            let errorHTML = "";
            if (j.error) {
                errorHTML = '<span class="queue-error">' + escapeHTML(j.error) + '</span>';
            }
            return '<div class="queue-item queue-' + j.status + '">' +
                '<span class="queue-status-dot"></span>' +
                '<span class="queue-url">' + escapeHTML(j.url) + '</span>' +
                '<span class="queue-status-label">' + statusLabel(j.status) + '</span>' +
                errorHTML +
                '</div>';
        }).join("");
    }

    function poll() {
        fetch("/api/queue")
            .then(r => r.json())
            .then(render)
            .catch(() => {});
    }

    // Initial render of server-side data, then start polling
    poll();
    setInterval(poll, 2000);
})();
