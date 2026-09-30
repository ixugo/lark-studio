package wails

import (
	"os/exec"
	"strings"
	"testing"
)

func TestYouTubeVerificationWaitsForTouchAndDownloadAddress(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("需要 Node 验证 WebView 脚本")
	}
	script := strings.NewReplacer("__NONCE__", `"nonce"`, "__VIDEO_ID__", `"gbetp6D7J_Q"`, "__QUALITY__", "1080").Replace(youtubeVerifyJS)
	harness := `const assert=require('node:assert/strict');let callback;const requests=[];const messages=[];
 global.location={origin:'https://embed.dlsrv.online'};
 global.window={_wails:{invoke:value=>messages.push(JSON.parse(value))}};window.top=window;
 global.navigator={userAgent:'test'};
 global.sessionStorage={getItem:()=> 'e30.'+Buffer.from(JSON.stringify({exp:Math.floor(Date.now()/1000)+300})).toString('base64url')+'.sig'};
 global.setInterval=fn=>{callback=fn;return 1};global.clearInterval=()=>{};
 global.fetch=async(path,options)=>{requests.push({path,body:JSON.parse(options.body)});if(path==='/api/session/touch'){assert.equal(messages.length,0);return{ok:true,status:200}};assert.equal(requests[0].path,'/api/session/touch');return{ok:true,status:200,json:async()=>({url:'https://yt1s-worker-5.dlsrv.online/tunnel?id=test'})}};
 ` + script + `;callback().then(()=>{assert.deepEqual(requests.map(r=>r.path),['/api/session/touch','/api/download/mp4']);assert.equal(messages.length,1);assert.equal(messages[0].address,'https://yt1s-worker-5.dlsrv.online/tunnel?id=test');assert.equal(requests[1].body.quality,'1080')}).catch(error=>{console.error(error);process.exitCode=1});`
	output, err := exec.CommandContext(t.Context(), node, "-e", harness).CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %s", err, output)
	}
}
