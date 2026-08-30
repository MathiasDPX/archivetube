window.addEventListener("beforeunload", function() {
    var video = document.querySelector('video');
    if (!video) return;

    var progresses = JSON.parse(window.localStorage.getItem("video_progresses"));
    if (!progresses || typeof progresses != "object") {
        progresses = {};
    }

    const videoId = video.dataset.videoId;
    const videoTs = video.currentTime

    if (videoId && videoTs != 0) {
        progresses[videoId] = videoTs;
        window.localStorage.setItem("video_progresses", JSON.stringify(progresses));
    }
})

document.addEventListener('DOMContentLoaded', function () {
    var video = document.querySelector('video');
    if (!video) return;

    const videoId = video.dataset.videoId;
    const channelId = video.dataset.channelId;

    // --- Audio track switching ---
    var audioSelect = document.getElementById('audio-track-select');
    var externalAudio = null;   // hidden <audio> element for non-original tracks
    var currentAudioUrl = null;

    function syncAudio() {
        if (!externalAudio) return;
        // Keep external audio in sync with the video
        if (Math.abs(externalAudio.currentTime - video.currentTime) > 0.3) {
            externalAudio.currentTime = video.currentTime;
        }
        if (video.paused) {
            externalAudio.pause();
        } else {
            externalAudio.play().catch(function () {});
        }
    }

    function switchAudioTrack(url) {
        if (url === 'original' || !url) {
            // Restore the video's native audio
            video.muted = false;
            if (externalAudio) {
                externalAudio.pause();
                externalAudio.src = '';
                externalAudio = null;
                currentAudioUrl = null;
            }
            return;
        }

        if (currentAudioUrl === url) return;
        currentAudioUrl = url;

        // Mute the video and play the external audio file
        video.muted = true;

        if (externalAudio) {
            externalAudio.pause();
        }
        externalAudio = new Audio(url);
        externalAudio.currentTime = video.currentTime;
        externalAudio.volume = video.volume;
        if (!video.paused) {
            externalAudio.play().catch(function () {});
        }

        // Sync events
        externalAudio.addEventListener('play', function () {
            syncAudio();
        });
    }

    if (audioSelect) {
        audioSelect.addEventListener('change', function () {
            switchAudioTrack(this.value);
        });

        // Keep external audio synced with video playback events
        video.addEventListener('play', function () {
            if (externalAudio) {
                externalAudio.play().catch(function () {});
            }
        });
        video.addEventListener('pause', function () {
            if (externalAudio) {
                externalAudio.pause();
            }
        });
        video.addEventListener('seeked', function () {
            if (externalAudio) {
                externalAudio.currentTime = video.currentTime;
            }
        });
        video.addEventListener('timeupdate', function () {
            if (externalAudio && Math.abs(externalAudio.currentTime - video.currentTime) > 0.5) {
                externalAudio.currentTime = video.currentTime;
            }
        });
        video.addEventListener('volumechange', function () {
            if (externalAudio) {
                externalAudio.volume = video.volume;
            }
        });
        video.addEventListener('ratechange', function () {
            if (externalAudio) {
                externalAudio.playbackRate = video.playbackRate;
            }
        });
    }

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

    if (params.get('t') == null) {
        var progresses = JSON.parse(window.localStorage.getItem("video_progresses"));
        progress = progresses[videoId];

        if (progress !== null && typeof progress !== 'undefined') {
            video.currentTime = progress;
        }
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
        const channelAvatarURL = document.getElementById('channel-avatar')?.src || '';

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

        // Seek will move 5s
        const seekValue = 5;
        // Change volume by 10% per press
        const volumeChange = 0.1;

        if (event.code == "Space" || event.key == "k") {
            // Play/pause
            if (video.paused) {
                if (playButton) playButton.setAttribute("data-icon", "u");
                video.play();
            } else {
                if (playButton) playButton.setAttribute("data-icon", "P");
                video.pause();
            }

        } else if (event.key == "f") {
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

        } else if (event.key == "m") {
            if (video.volume != 0) {
                lastMuteVolume = video.volume;
                video.volume = 0;
            } else {
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
