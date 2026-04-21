# Dearrow

![Screenshot of four videos with Dearrow disabled](../assets/images/normal_videos.png)

<center> Without Dearrow ⬆️ // ⬇️ With Dearrow</center>

![Screenshot of four videos with Dearrow enabled](../assets/images/dearrow_videos.png)

ArchiveTube supports <a href="https://dearrow.ajay.app/" target="_blank">Dearrow</a>, an open source extension for crowdsourcing better titles and thumbnails. Using Dearrow does not change current/future archived thumbnails and titles but change the images returned by `/data/media/channels/channel-id/video-id/video.webp`

```toml
[dearrow]
enable = true
```

Additionally, you can add a `main_api` and `thumb_api` variable to change the API used. The Thumbnail API's source code is available on <a href="https://github.com/ajayyy/DeArrowThumbnailCache" target="_blank">ajayyy/DeArrowThumbnailCache</a>

Defaults: 
```toml
main_api = "https://sponsor.ajay.app"
thumb_api = "https://dearrow-thumb.ajay.app"
```

> [!IMPORTANT] 
> You need to remove any trailing slash in URLs