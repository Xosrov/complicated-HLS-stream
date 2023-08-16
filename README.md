# Making HLS Streams More Annoying to Download

Grabbing HLS streams is mad easy if you have the manifest file. Some websites try to stop this by requiring headers or using POST requests instead of GET but that's child's play for downloading tools. They just copy the headers or emulate the POST request. Lame!

This repo is all about coming up with sneaky ways to make grabbing HLS streams a giant pain, without using actual DRM (because that sucks). Like a challenge to see how far we can take things before Average Joe can't just download streams with IDM anymore.

Obviously a determined hacker could still reverse engineer the JavaScript and automate things. So if you really don't want your content stolen, pay up for the commercial DRM big dogs. But even that can be cracked if someone tries hard enough.

## Tricks Used
Here's a quick rundown of the tricks I used to trip up automated HLS downloaders. Of course, someone could still write an extension or code to emulate everything done here. But it adds more hoops to jump through.

### "Hide" the Master Manifest
It's 2023 and browser dev tools still don't show WebRTC messages. Perfect for sneaking the master manifest and variants! We grab all the HLS pieces through WebRTC and construct the stream from there.

### Scramble Fragment Names
Got fragments named `1.ts`, `2.ts`, etc? Might as well put up a neon sign saying "Steal me!"

### Encrypt Errything
Unencrypted fragments? Weak sauce. Same key for all fragments? I expected more. Instead, let's encrypt each segment with a unique key!

### GET Outta Here
Still using GET requests in 2023? Borrrring! Let's fetch fragments with POST instead!

## Usage

1. Install `python`, `golang` and `ffmpeg`.
2. Generate segments for your video: `bash generate.sh [video_file] [ip_addr]`
3. Run server: `go run main.go`

EZ right?