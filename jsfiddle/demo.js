/* eslint-env browser */

function logWithTimestamp(message) {
	const now = new Date();
	console.log(`${now.toISOString()}: ${message}`);
}

// logWithTimestamp("This is a message.");  // Outputs: "2023-12-12T20:44:29.123Z: This is a message."


let pc = new RTCPeerConnection({
	iceServers: [
		{
			// urls: ["stun:stun1.l.google.com:19302", "stun:stun2.l.google.com:19302", "stun:stun.l.google.com:19302", "stun:stun3.l.google.com:19302", "stun:stun4.l.google.com:19302"]
			// urls: 'turn:192.168.1.8:3478',username: 'foo',  credential: 'bar'
			urls: 'turn:23.105.204.193:13478',username: 'foo',  credential: 'bar'
			// iceServers: [{urls: 'turn:192.168.1.8:3478',username: 'foo',  credential: 'bar'}]
		}
	]
})
let log = msg => {
	document.getElementById('div').innerHTML += msg + '<br>'
}


let isDataChannelReady = false;

window.onload = function() {

	let localStream;
	pc.ontrack = function (event) {
		var el = document.createElement(event.track.kind)
		el.srcObject = event.streams[0]
		el.autoplay = true
		el.controls = true

		document.getElementById('remoteVideos').appendChild(el)
	}

	pc.oniceconnectionstatechange = e => {
		log(pc.iceConnectionState)
		if (['closed', 'failed', 'disconnected'].includes(pc.iceConnectionState)) {
			closeConnection();
		}
	}
	pc.onicecandidate = event => {
		if (event.candidate === null) {
			document.getElementById('localSessionDescription').value = btoa(JSON.stringify(pc.localDescription))
		}
	}
	navigator.mediaDevices.getUserMedia({ video: false, audio: true })
		.then(stream => {
			localStream=stream
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
					// logWithTimestamp(answer);	
					//pc.setRemoteDescription(answer);

					if (answer === '') {
						alert('Session Description must not be empty')
					}
					try {
						//pc.setRemoteDescription(JSON.parse(answer));
						logWithTimestamp(answer);
						pc.setRemoteDescription(answer);
					} catch (e) {
						alert(e)
					}
				});
			});

		}).catch(log)
	document.getElementById('disconnectButton').addEventListener('click', () => {
		closeConnection()
	});

	logWithTimestamp("createDataChannel now ")
	let dc = pc.createDataChannel("onetext");
	dc.onmessage = function (event) {
		logWithTimestamp(event.data);
		let messageList = document.getElementById('message-list');
		let newMessage = document.createElement('li');
		newMessage.textContent = event.data;
		messageList.appendChild(newMessage);
	};

	dc.onopen = function () {
		logWithTimestamp("Data channel is open and ready to be used.");
		isDataChannelReady = true;
	};

	dc.onerror = function (error) {
		logWithTimestamp("Data Channel Error:", error);
	};

	document.getElementById('send-button').addEventListener('click', function() {
		let messageInput = document.getElementById('message-input');
		let message = messageInput.value;
		messageInput.value = '';

		logWithTimestamp("Sending message: " + message);
		if (isDataChannelReady) {
			dc.send(message);
		} else {
			logWithTimestamp("Data channel is not ready yet.");
		}
	});

}

function closeConnection() {
	logWithTimestamp('closing connection');
	if (pc) {
		pc.close();
		pc = null;
	}

	if (localStream) {
		localStream.getTracks().forEach(track => track.stop());
		localStream = null;
	}
}

