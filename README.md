# WebRTC-WAV-to-Disk
An example of saving a WAV file(pcm audio from browser microphone) to disk using WebRTC.

## How these work
- webrtc js must run on https, so we need config a https domain, here is an enginx config sample
- frontend as client record audio and encode as pcm stream then send to server
- golang server specify sdp only use pcm codec and save pcm to disk
- turn/stun issue, we use `github.com/pion/turn` which provide stun and turn service

[webrtcforthecurious.com](https://webrtcforthecurious.com/docs/01-what-why-and-how/) this blog is practical, and I suggest reading it first ,
then run this demo ,
also there are a lot of examples [github.com/pion/example-webrtc-applications](github.com/pion/example-webrtc-applications) then take a look and do the excersise
