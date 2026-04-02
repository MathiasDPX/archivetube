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

    const videoTitle = document.getElementById('video-title').innerText;
    const channelName = document.getElementById('channel-name').innerText;
    const channelAvatarURL = document.getElementById('channel-avatar').src;

    const videoId = video['data-video-id']
    const channelId = video['data-channel-id']

    if ('mediaSession' in navigator) {
        metadata = {
            title: videoTitle,
            artist: channelName,
            artwork: [
                {
                    src: channelAvatarURL,
                    sizes: "512x512" // lie
                },
                {
                    src: `/data/media/channels/${channelId}/${videoId}/video.webp`,
                    sizes: "1920x1080" // lie again
                }
            ]
        }
        console.log("Starting mediaSession with", metadata);
        navigator.mediaSession.metadata = new MediaMetadata(metadata);

        // player to mediaSession events
        video.addEventListener('play', function () {
            navigator.mediaSession.playbackState = "playing";
        })

        video.addEventListener('pause', function () {
            navigator.mediaSession.playbackState = "paused";
        })

        video.addEventListener('ended', function () {
            navigator.mediaSession.playbackState = "none";
        })

        // mediaSession to player events
        navigator.mediaSession.setActionHandler("play", () => {
            video.play();
        });

        navigator.mediaSession.setActionHandler("pause", () => {
            video.pause();
        });

        setInterval(() => {
            navigator.mediaSession.setPositionState({
                duration: video.duration,
                playbackRate: video.playbackRate,
                position: video.currentTime
            })
        }, 1000);
    } else {
        console.warn("mediaSession unavailable on this browser :'(")
    }
});
