window.addEventListener("beforeunload", function() {
    var player = videojs.getPlayer('video-player');
    if (!player) return;

    var progresses = JSON.parse(window.localStorage.getItem("video_progresses"));
    if (!progresses || typeof progresses != "object") {
        progresses = {};
    }

    const videoId = player.el().querySelector('video').dataset.videoId;
    const videoTs = player.currentTime();

    if (videoId && videoTs != 0) {
        progresses[videoId] = videoTs;
        window.localStorage.setItem("video_progresses", JSON.stringify(progresses));
    }
});

document.addEventListener('DOMContentLoaded', function () {
    var videoEl = document.getElementById('video-player');
    if (!videoEl) return;

    const videoId = videoEl.dataset.videoId;
    const channelId = videoEl.dataset.channelId;

    var player = videojs('video-player', {
        fluid: true,
        playbackRates: [0.5, 1, 1.25, 1.5, 2],
        controlBar: {
            pictureInPictureToggle: false
        },
        userActions: {
            hotkeys: false
        }
    });

    // --- Audio tracks integrated into Video.js ---
    var audioTracksData = [];
    try {
        audioTracksData = JSON.parse(videoEl.dataset.audioTracks || '[]');
    } catch (e) {
        console.warn('Failed to parse audio tracks data', e);
    }
    var externalAudio = null;

    if (audioTracksData.length > 1) {
        var trackList = player.audioTracks();
        var trackMap = {};  // trackId -> url

        audioTracksData.forEach(function (t, i) {
            var url = t.url;
            var label = t.label;
            var isOriginal = t.isOriginal;
            var trackId = isOriginal ? 'original' : 'audio-track-' + i;

            trackMap[trackId] = url;

            var track = new videojs.AudioTrack({
                id: trackId,
                kind: isOriginal ? 'main' : 'translation',
                label: label,
                language: isOriginal ? '' : (t.lang || '')
            });

            if (isOriginal) {
                track.enabled = true;
            }

            trackList.addTrack(track);
        });

        function switchAudioTrack(url) {
            if (url === 'original' || !url) {
                player.muted(false);
                if (externalAudio) {
                    externalAudio.pause();
                    externalAudio.src = '';
                    externalAudio = null;
                }
                return;
            }

            player.muted(true);

            if (externalAudio) {
                externalAudio.pause();
            }
            externalAudio = new Audio(url);
            externalAudio.currentTime = player.currentTime();
            externalAudio.volume = player.volume();
            if (!player.paused()) {
                externalAudio.play().catch(function () {});
            }
        }

        trackList.addEventListener('change', function () {
            for (var i = 0; i < trackList.length; i++) {
                if (trackList[i].enabled) {
                    switchAudioTrack(trackMap[trackList[i].id]);
                    break;
                }
            }
        });

        // Keep external audio synced with player events
        player.on('play', function () {
            if (externalAudio) {
                externalAudio.play().catch(function () {});
            }
        });
        player.on('pause', function () {
            if (externalAudio) {
                externalAudio.pause();
            }
        });
        player.on('seeked', function () {
            if (externalAudio) {
                externalAudio.currentTime = player.currentTime();
            }
        });
        player.on('timeupdate', function () {
            if (externalAudio && Math.abs(externalAudio.currentTime - player.currentTime()) > 0.5) {
                externalAudio.currentTime = player.currentTime();
            }
        });
        player.on('volumechange', function () {
            if (externalAudio) {
                externalAudio.volume = player.volume();
            }
        });
        player.on('ratechange', function () {
            if (externalAudio) {
                externalAudio.playbackRate = player.playbackRate();
            }
        });
    }

    let params = new URLSearchParams(document.location.search);
    var chapters = Array.from(document.querySelectorAll('.chapter-item'));

    chapters.forEach(function (item) {
        item.addEventListener('click', function () {
            player.currentTime(parseFloat(this.dataset.time));
            player.play();
        });
    });

    if (chapters.length > 0) {
        player.on('timeupdate', function () {
            var t = player.currentTime();
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
            player.one('loadedmetadata', function () {
                player.currentTime(progress);
            });
        }
    }

    try {
        const timestamp = params.get('t');
        if (timestamp != null) {
            player.one('loadedmetadata', function () {
                player.currentTime(parseFloat(timestamp));
            });
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
                    sizes: "1920x1080"
                },
                {
                    src: channelAvatarURL,
                    sizes: "512x512"
                }
            ]
        }
        console.log("Starting mediaSession with", metadata);
        navigator.mediaSession.metadata = new MediaMetadata(metadata);

        player.on('play', function () {
            navigator.mediaSession.playbackState = "playing";
        });

        player.on('pause', function () {
            navigator.mediaSession.playbackState = "paused";
        });

        player.on('ended', function () {
            navigator.mediaSession.playbackState = "none";
        });

        navigator.mediaSession.setActionHandler("play", function () {
            player.play();
        });

        navigator.mediaSession.setActionHandler("pause", function () {
            player.pause();
        });

        setInterval(function () {
            navigator.mediaSession.setPositionState({
                duration: player.duration(),
                playbackRate: player.playbackRate(),
                position: player.currentTime()
            });
        }, 1000);
    } else {
        console.warn("mediaSession unavailable on this browser :'(")
    }

    let lastMuteVolume = player.volume();

    addEventListener("keydown", function (event) {
        const tag = event.target && event.target.tagName;
        if (tag === "INPUT" || tag === "TEXTAREA" || (event.target && event.target.isContentEditable)) {
            return;
        }

        const seekValue = 5;
        const volumeChange = 0.1;

        if (event.code == "Space" || event.key == "k") {
            if (player.paused()) {
                player.play();
            } else {
                player.pause();
            }

        } else if (event.key == "f") {
            if (player.isFullscreen()) {
                player.exitFullscreen();
            } else {
                player.requestFullscreen();
            }

        } else if (event.code == "ArrowLeft") {
            player.currentTime(player.currentTime() - seekValue);

        } else if (event.code == "ArrowRight") {
            player.currentTime(player.currentTime() + seekValue);

        } else if (event.code.startsWith("Digit")) {
            percent = parseInt(event.code.slice(5)) / 10;
            player.currentTime(player.duration() * percent);

        } else if (event.code == "ArrowUp") {
            player.volume(Math.min(1, player.volume() + volumeChange));

        } else if (event.code == "ArrowDown") {
            player.volume(Math.max(0, player.volume() - volumeChange));

        } else if (event.key == "m") {
            if (player.volume() != 0) {
                lastMuteVolume = player.volume();
                player.volume(0);
            } else {
                if (lastMuteVolume == 0) {
                    player.volume(1);
                } else {
                    player.volume(lastMuteVolume);
                }
            }

        } else {
            return;
        }

        event.preventDefault();
    });
});
