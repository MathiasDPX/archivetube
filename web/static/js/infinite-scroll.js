const grid = document.querySelector('[data-infinite-grid]');

const apiUrl = grid.dataset.infiniteApi;
const showChannel = grid.dataset.infiniteShowChannel !== 'false';
let page = parseInt(grid.dataset.infinitePage, 10) || 1;
let total = parseInt(grid.dataset.infiniteTotal, 10) || 0;
let perPage = parseInt(grid.dataset.infinitePerpage, 10) || 24;
let loading = false;

const sentinel = document.createElement('div');
sentinel.className = 'infinite-sentinel';
grid.parentNode.insertBefore(sentinel, grid.nextSibling);

function hasMore() {
    return page * perPage < total;
}

function fmtDuration(seconds) {
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = seconds % 60;
    if (h > 0) return h + ':' + String(m).padStart(2, '0') + ':' + String(s).padStart(2, '0');
    return m + ':' + String(s).padStart(2, '0');
}

function fmtDate(dateStr) {
    const d = new Date(dateStr);
    return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

function webPath(p) {
    if (!p) return '';
    return p.replace(/\\/g, '/').replace(/^\/+/, '');
}

function createCard(v) {
    const card = document.createElement('div');
    card.className = 'video-card';

    let thumbImg = '';
    if (v.ThumbnailRelPath) {
        thumbImg = '<img src="/data/' + webPath(v.ThumbnailRelPath) + '" alt="" loading="lazy">';
    }

    let channelHtml = '';
    if (showChannel && v.ChannelName) {
        channelHtml = '<a href="/creators/' + v.ChannelYoutubeID + '" class="card-channel">' + v.ChannelName + '</a>';
    }

    card.innerHTML =
        '<a href="/videos/' + v.YoutubeVideoID + '" class="thumb-link">' +
        '<div class="thumb">' +
        thumbImg +
        '<span class="badge-duration">' + fmtDuration(v.DurationSeconds) + '</span>' +
        '</div>' +
        '</a>' +
        '<div class="card-body">' +
        '<a href="/videos/' + v.YoutubeVideoID + '" class="card-title">' + v.Title + '</a>' +
        channelHtml +
        '<span class="card-meta">Archived ' + fmtDate(v.ArchivedAt) + '</span>' +
        '</div>';

    return card;
}

function loadMore() {
    if (loading || !hasMore()) return;
    loading = true;

    const nextPage = page + 1;
    const sep = apiUrl.includes('?') ? '&' : '?';
    const url = apiUrl + sep + 'page=' + nextPage;

    fetch(url)
        .then(function (res) { return res.json(); })
        .then(function (data) {
            if (data.videos && data.videos.length > 0) {
                data.videos.forEach(function (v) {
                    grid.appendChild(createCard(v));
                });
                page = data.page;
                total = data.total;
                perPage = data.perPage;
            }
            loading = false;
        })
        .catch(function () {
            loading = false;
        });
}

if (hasMore()) {
    const observer = new IntersectionObserver(function (entries) {
        if (entries[0].isIntersecting) {
            loadMore();
        }
    }, { rootMargin: '400px' });
    observer.observe(sentinel);
}