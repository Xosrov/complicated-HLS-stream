"use strict"

class fragmentLoader extends Hls.DefaultConfig.loader {
    constructor(config) {
        super(config);
        var load = this.load.bind(this);
        this.load = function (context, config, callbacks) {
            // override frag request
            console.log("frag req");
            fetch("/frag", {
                method: 'post',
                headers: {
                    'Accept': 'application/octet-stream, */*',
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    "name": context.url.substring(5),
                })
            })
                .then(resp => {
                    resp.blob().then(blob => {
                        const blobUrl = URL.createObjectURL(blob);
                        context.url = blobUrl;
                        load(context, config, callbacks);
                    })
                        .catch(err => {
                            console.log("get blob err")
                        })
                })
                .catch(err => {
                    console.log("req err")
                })
        }
    }
}

function combineArrayBuffers(arrayBuffers) {
    // Calculate the total length of the combined ArrayBuffers
    let totalLength = 0;
    for (const arrayBuffer of arrayBuffers) {
        totalLength += arrayBuffer.byteLength;
    }

    // Create a new ArrayBuffer with the calculated total length
    const combinedBuffer = new ArrayBuffer(totalLength);

    // Create a Uint8Array view for the new ArrayBuffer
    const combinedArray = new Uint8Array(combinedBuffer);

    // Copy the contents of each ArrayBuffer into the combinedArray
    let offset = 0;
    for (const arrayBuffer of arrayBuffers) {
        const sourceArray = new Uint8Array(arrayBuffer);
        combinedArray.set(sourceArray, offset);
        offset += sourceArray.length;
    }

    return combinedBuffer;
}
var hls;
function changeQuality(lvl) {
    hls.currentLevel = lvl;
}

function gotManifest() {
    console.log("process manifest");
    var enc = new TextDecoder("utf-8");
    var decodedMaster = enc.decode(fullMaster);
    for (const [variantName, variantData] of Object.entries(fullVariants)) {
        var decodedVariantData = enc.decode(variantData);
        // create blob for variant data
        const blob = new Blob([decodedVariantData], { type: 'text/plain' });
        const blobUrl = URL.createObjectURL(blob);
        // replace this with the manifest url
        decodedMaster = decodedMaster.replace(variantName, blobUrl);
    }
    console.log(decodedMaster)
    // create blob for new master
    const blob = new Blob([decodedMaster], { type: 'text/plain' });
    const blobUrl = URL.createObjectURL(blob);
    var video = document.getElementById('video');
    if (Hls.isSupported()) {
        hls = new Hls({
            fLoader: fragmentLoader,
        });
        hls.attachMedia(video);
        hls.on(Hls.Events.MEDIA_ATTACHED, function () {
            console.log('bound hls to DOM element');
            hls.loadSource(blobUrl);
            hls.on(Hls.Events.MANIFEST_PARSED, function (event, data) {
                console.log('manifest loaded with ' + data.levels.length + ' quality level(s)');
            });
        });
        hls.on(Hls.Events.ERROR, function (event, data) {
            var errorType = data.type;
            var errorDetails = data.details;
            var errorFatal = data.fatal;
            switch (data.details) {
                case Hls.ErrorDetails.FRAG_LOAD_ERROR:
                    console.log('error: FRAG_LOAD_ERROR');
                    break;
                case Hls.ErrorDetails.MEDIA_ERROR:
                    console.log('error: MEDIA_ERROR');
                    break;
                case Hls.ErrorDetails.OTHER_ERROR:
                    console.log('error" OTHER_ERROR');
                    break;
                default:
                    console.log("ERR");
                    console.log(event)
                    console.log(data)
                    break;
            }
        });
    }
}

var currentType = ""
var masterBuffer = [];
var variantBuffer = [];

var fullMaster;
var fullVariants = {};
function onDcMessage(msg) {
    if (typeof msg.data === 'string') {
        if (msg.data.startsWith("MASTER")) {
            currentType = msg.data;
            console.log("getting master")
            masterBuffer = [];
        } else if (msg.data.startsWith("VARIANT_")) {
            currentType = msg.data;
            console.log("getting variant " + msg.data.substring(8))
            variantBuffer = [];
        } else if (msg.data.startsWith("ERR")) {
            console.log("err")
        } else if (msg.data.startsWith("END")) {
            if (currentType === "MASTER") {
                console.log("got" + currentType)
                fullMaster = combineArrayBuffers(masterBuffer);
            } else if (currentType.startsWith("VARIANT_")) {
                console.log("got" + currentType)
                fullVariants[currentType.substring(8)] = combineArrayBuffers(variantBuffer);
            }
        } else if (msg.data === "CLOSING") {
            // start
            gotManifest();
        }
    } else {
        if (currentType === "MASTER") {
            masterBuffer.push(msg.data);
        } else if (currentType.startsWith("VARIANT_")) {
            variantBuffer.push(msg.data);
        }
    }
}

window.onload = (event) => {
    let pc = new RTCPeerConnection()
    var dc = pc.createDataChannel('data', {
        maxRetransmits: null,
        ordered: true,
    })

    dc.onmessage = onDcMessage;

    pc.oniceconnectionstatechange = () => {
        let el = document.createElement('p')
        el.appendChild(document.createTextNode(pc.iceConnectionState))
        document.getElementById('iceConnectionStates').appendChild(el);
    }

    pc.createOffer()
        .then(offer => {
            pc.setLocalDescription(offer)

            return fetch(`/signal`, {
                method: 'post',
                headers: {
                    'Accept': 'application/json, text/plain, */*',
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(offer)
            })
        })
        .then(res => res.json())
        .then(res => {
            pc.setRemoteDescription(res)
        })
        .catch(alert)
};