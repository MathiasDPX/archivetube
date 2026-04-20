document.addEventListener('DOMContentLoaded', function () {
    var video = document.querySelector('video');
    if (!video) return;

    let params = new URLSearchParams(document.location.search);
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

    try {
        const timestamp = params.get('t');
        if (timestamp != null) {
            video.currentTime = parseFloat(timestamp);
        }
    } catch (error) {
        console.warn("Failed to process timecode parameter")
    }

    if ('mediaSession' in navigator) {
        const videoTitle = document.getElementById('video-title').innerText;
        const channelName = document.getElementById('channel-name').innerText;
        const channelAvatarURL = document.getElementById('channel-avatar').src;

        const videoId = video['data-video-id']
        const channelId = video['data-channel-id']

        metadata = {
            title: videoTitle,
            artist: channelName,
            artwork: [
                {
                    src: `/data/media/channels/${channelId}/${videoId}/video.webp`,
                    sizes: "1920x1080" // lie again
                },
                {
                    src: channelAvatarURL,
                    sizes: "512x512" // lie
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

    const playButton = document.getElementById("play");
    let lastMuteVolume = video.volume;

    addEventListener("keydown", (event) => {
        const tag = event.target && event.target.tagName;
        if (tag === "INPUT" || tag === "TEXTAREA" || (event.target && event.target.isContentEditable)) {
            return;
        }

        console.log("debug: ", event.code)

        // Seek will move 5s
        const seekValue = 5;
        // Change volume by 10% per press
        const volumeChange = 0.1;

        if (event.code == "Space" || event.code == "KeyK") {
            // Play/pause
            if (video.paused) {
                if (playButton) playButton.setAttribute("data-icon", "u");
                video.play();
            } else {
                if (playButton) playButton.setAttribute("data-icon", "P");
                video.pause();
            }
        } else if (event.code == "KeyF") {
            // Fullscreen
            if (document.fullscreenElement != null) {
                document.exitFullscreen()
            } else {
                video.requestFullscreen()
            }
        } else if (event.code == "ArrowLeft") {
            video.currentTime -= seekValue;
        } else if (event.code == "ArrowRight") {
            video.currentTime += seekValue;
        } else if (event.code.startsWith("Digit")) {
            // todo: same for "NumpadX"
            percent = parseInt(event.code.slice(5)) / 10;
            video.currentTime = video.duration * percent;
        } else if (event.code == "ArrowUp") {
            video.volume = Math.min(1, video.volume + volumeChange);
        } else if (event.code == "ArrowDown") {
            video.volume = Math.max(0, video.volume - volumeChange);
        } else if (event.code == "KeyM") {
            if (video.volume != 0) {
                lastMuteVolume = video.volume;
                console.log("set lastmutevolume to "+lastMuteVolume)
                video.volume = 0;
            } else {
                console.log(lastMuteVolume)
                if (lastMuteVolume == 0) {
                    video.volume = 1;
                } else {
                    video.volume = lastMuteVolume;
                }
            }
        } else {
            return;
        }

        event.preventDefault();
    })
});
