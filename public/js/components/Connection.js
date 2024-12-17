import { baseUrl } from '../config.js';

export default class Connection {
	constructor(token, onOpen = () => {}) {
		this.socket = new WebSocket(baseUrl + '/ws?token=' + token);
		this.callbacks = {};

		this.socket.addEventListener('open', onOpen);

		this.socket.addEventListener('message', (event) => {
			event.data.split('\n').forEach(part => {
				if (part.length > 0) {
					const message = JSON.parse(part);
					console.log(message);
					if (this.callbacks[message.type]) {
					  this.callbacks[message.type].forEach(cb => cb(message));
					}
				}
			});
		});
	}

	on(event, callback) {
		if (!this.callbacks[event]) {
			this.callbacks[event] = [];
		}
		this.callbacks[event].push(callback);
	}

	off(event, callback) {
		if (this.callbacks[event]) {
			this.callbacks[event] = this.callbacks[event].filter(cb => cb !== callback);
		}
	}

	send(message) {
		this.socket.send(JSON.stringify(message));
	}
}