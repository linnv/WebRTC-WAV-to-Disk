# WebRTC-WAV-to-Disk
An example of saving a WAV file(pcm audio from browser microphone) to disk using WebRTC.

## How these work
- webrtc js must run on https, so we need config a https domain, here is an enginx config sample
- frontend as client record audio and encode as pcm stream then send to server
- golang server specify sdp only use pcm codec and save pcm to disk
- turn/stun issue, we use `github.com/pion/turn` which provide stun and turn service

[webrtcforthecurious.com](https://webrtcforthecurious.com/docs/01-what-why-and-how/) this blog is practical, and I suggest reading it first, then run this demo, also there are a lot of examples [github.com/pion/example-webrtc-applications](github.com/pion/example-webrtc-applications) then take a look and do the excersise


gen self-cert
```
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout webrtc.devinner.key -out webrtc.devinner.crt
```

nginx config
```
#cat   /opt/homebrew/etc/nginx/servers/webrtcinner.conf
server {
    server_name webrtc.devinner;

    listen 443 ssl; 

    ssl_certificate /Users/jialinwu/qn-pc/nginx/webrtc.devinner.crt; #you should click the crt and adding it to system trust in macOS
    ssl_certificate_key /Users/jialinwu/qn-pc/nginx/webrtc.devinner.key;
   location / {
        proxy_pass http://127.0.0.1:8013;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}

server {
    if ($host = webrtc.devinner) {
        return 301 https://$host$request_uri;
    } 


    listen       80;
    server_name  webrtc.devinner;
    return 404; 
}
