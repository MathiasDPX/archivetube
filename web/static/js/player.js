document.addEventListener('DOMContentLoaded', function () {
    var video = document.getElementById('video-player');
    if (!video) return;

    var chapters = Array.from(document.querySelectorAll('.chapter-item'));

    chapters.forEach(function (item) {
        item.addEventListener('click', function () {
            video.currentTime = parseFloat(this.dataset.time);
            video.play();
        });
    });

    if (chapters.length > 0) {
        video.addEventListener('timeupdate', function () {
            var t = video.currentTime;
            chapters.forEach(function (ch) {
                var start = parseFloat(ch.dataset.time);
                var end = parseFloat(ch.dataset.end);
                ch.classList.toggle('active', t >= start && t < end);
            });
        });
    }
});
