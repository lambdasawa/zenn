.PHONY: server
server:
	cd go/goja && npm i && npx esbuild index.js --bundle --minify --target=es2017 >dist.js
	cd go && go run .

.PHONY: client
client:
	cd js/my-app && npm i && npm start
