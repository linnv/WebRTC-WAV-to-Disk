/* eslint-env browser */

let pc = new RTCPeerConnection({
	iceServers: [
		{
			urls: ["stun:stun1.l.google.com:19302", "stun:stun2.l.google.com:19302", "stun:stun.l.google.com:19302", "stun:stun3.l.google.com:19302", "stun:stun4.l.google.com:19302"]
			// urls: 'turn:192.168.1.8:3478',username: 'foo',  credential: 'bar'
			// iceServers: [{urls: 'turn:192.168.1.8:3478',username: 'foo',  credential: 'bar'}]
		}
	]
})
let log = msg => {
	document.getElementById('div').innerHTML += msg + '<br>'
}

pc.ontrack = function (event) {
	var el = document.createElement(event.track.kind)
	el.srcObject = event.streams[0]
	el.autoplay = true
	el.controls = true

	document.getElementById('remoteVideos').appendChild(el)
}

pc.oniceconnectionstatechange = e => log(pc.iceConnectionState)
pc.onicecandidate = event => {
	if (event.candidate === null) {
		document.getElementById('localSessionDescription').value = btoa(JSON.stringify(pc.localDescription))
	}
}

// Offer to receive 1 audio, and 1 video track
pc.addTransceiver('video', {'direction': 'sendrecv'})
pc.addTransceiver('audio', {'direction': 'sendrecv'})

// pc.createOffer().then(d => pc.setLocalDescription(d)).catch(log)

// window.startSession = () => {
//   let sd = document.getElementById('remoteSessionDescription').value
//   if (sd === '') {
//     return alert('Session Description must not be empty')
//   }
//
//   try {
//     pc.setRemoteDescription(JSON.parse(atob(sd)))
//   } catch (e) {
//     alert(e)
//   }
// }

navigator.mediaDevices.getUserMedia({ video: false, audio: true })
	.then(stream => {
		stream.getTracks().forEach(track => pc.addTrack(track, stream))
		// stream.getTracks().forEach(function(track) {
		// 	pc.addTrack(track, stream);
		// });

		// displayVideo(stream)
		// pc.createOffer().then(d => pc.setLocalDescription(d)).catch(log)

		pc.createOffer().then(offer => {
			pc.setLocalDescription(offer).catch(log);
			// Send the offer to the server
			// fetch('http://127.0.0.1:8013/offer', {
			// fetch('https://webrtc.jialinwu.com/offer', {
			fetch('/offer', {
				method: 'POST',
				body: JSON.stringify(offer),
				headers: {'Content-Type': 'application/json'}
			}).then(response => response.json()).then(answer => {
				// Set the SDP answer
				console.log(answer);	
				//pc.setRemoteDescription(answer);

				if (answer === '') {
					alert('Session Description must not be empty')
				}
				try {
					//pc.setRemoteDescription(JSON.parse(answer));
					pc.setRemoteDescription(answer);
				} catch (e) {
					alert(e)
				}
			});
		});

	}).catch(log)

function keepOnlyCodecs(sdp, codecsToKeep) {
	let lines = sdp.split('\n');

	let audioMLineIndex = null;
	for (let i = 0; i < lines.length; i++) {
		if (lines[i].startsWith('m=audio')) {
			audioMLineIndex = i;
			break;
		}
	}

	if (audioMLineIndex === null) {
		console.log('No m=audio line found in SDP');
		return sdp;
	}

	let mLineParts = lines[audioMLineIndex].split(' ');
	let payloadTypesToKeep = [];
	for (let codec of codecsToKeep) {
		for (let i = 3; i < mLineParts.length; i++) {
			if (lines.some(line => line.startsWith('a=rtpmap:' + mLineParts[i] + ' ' + codec + '/'))) {
				payloadTypesToKeep.push(mLineParts[i]);
				break;
			}
		}
	}

	if (payloadTypesToKeep.length === 0) {
		console.log('No payload types found in SDP for codecs: ' + codecsToKeep.join(', '));
		return sdp;
	}

	let newLines = lines.filter(line => {
		return payloadTypesToKeep.some(payloadType =>
			line.startsWith('a=rtpmap:' + payloadType + ' ') ||
			line.startsWith('a=rtcp-fb:' + payloadType) ||
			line.startsWith('a=fmtp:' + payloadType)
		) || !line.startsWith('a=rtpmap:') && !line.startsWith('a=rtcp-fb:') && !line.startsWith('a=fmtp:');
	});

	let newMLineParts = mLineParts.filter(part => payloadTypesToKeep.includes(part));
	newLines[audioMLineIndex] = newMLineParts.join(' ');

	return newLines.join('\n');
}
function removeCodecs(sdp, codecsToRemove) {
	let lines = sdp.split('\n');

	let audioMLineIndex = null;
	for (let i = 0; i < lines.length; i++) {
		if (lines[i].startsWith('m=audio')) {
			audioMLineIndex = i;
			break;
		}
	}

	if (audioMLineIndex === null) {
		console.log('No m=audio line found in SDP');
		return sdp;
	}

	let mLineParts = lines[audioMLineIndex].split(' ');
	let payloadTypesToRemove = [];
	for (let codec of codecsToRemove) {
		for (let i = 3; i < mLineParts.length; i++) {
			if (lines.some(line => line.startsWith('a=rtpmap:' + mLineParts[i] + ' ' + codec + '/'))) {
				payloadTypesToRemove.push(mLineParts[i]);
				break;
			}
		}
	}

	if (payloadTypesToRemove.length === 0) {
		console.log('No payload types found in SDP for codecs: ' + codecsToRemove.join(', '));
		return sdp;
	}

	let newLines = lines.filter(line => {
		return !payloadTypesToRemove.some(payloadType =>
			line.startsWith('a=rtpmap:' + payloadType + ' ') ||
			line.startsWith('a=rtcp-fb:' + payloadType) ||
			line.startsWith('a=fmtp:' + payloadType)
		);
	});

	let newMLineParts = mLineParts.filter(part => !payloadTypesToRemove.includes(part));
	newLines[audioMLineIndex] = newMLineParts.join(' ');

	return newLines.join('\n');
}


window.startSession = () => {
	pc.createOffer().then(offer => {
		pc.setLocalDescription(offer).catch(log);
		// Send the offer to the server
		fetch('http://127.0.0.1:8013/offer', {
			method: 'POST',
			body: JSON.stringify(offer),
			headers: {'Content-Type': 'application/json'}
		}).then(response => response.json()).then(answer => {
			// Set the SDP answer
			console.log(answer);	
			//pc.setRemoteDescription(answer);

			if (answer === '') {
				alert('Session Description must not be empty')
			}
			try {
				//pc.setRemoteDescription(JSON.parse(answer));
				pc.setRemoteDescription(answer);
			} catch (e) {
				alert(e)
			}
		});
	});
}
