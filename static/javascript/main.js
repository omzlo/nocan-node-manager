
function display_nodes() {
    $.ajax({
        url: "/api/nodes"
    })
    .done(function(data) {
        var nodeCount = data.length
        for (var i = 0; i<nodeCount; i++) {
            $.ajax({
                url: "/api/nodes/" + data[i]
            })
            .done(function(xdata) {
                node = $("<div/>", {"class": "widget"})
                node.append($("<p/>")
                    .addClass("widget-item")
                    .append($("<strong/>")
                        .text(xdata.udid)
                    )
                )
                node.append($("<p/>", {"class": "widget-item"}).text("Last seen: " + xdata.last_seen))
                $("#nodes").append(node)
            });
        }
    });
}


function update_progress() {
    if (this.readyState == 4) {
        if (this.status==202) { // accepted
            var poll_id = setInterval(request, 500);
            var theLocation = this.getResponseHeader("Location");

            //console.log("Going to " + theLocation)
            function request() {
                var xhttp2 = Ajax();
                xhttp2.onreadystatechange = function() {
                    if (this.readyState == 4) {
                        if (this.status==200) {
                            //console.log("Response: " + this.responseText);
                            if (this.responseText == "done") {
                                clearInterval(poll_id);
                                document.getElementById("job_progress").innerHTML = "done";
                                if (xhttp2.getResponseHeader("Location")) {
                                    //console.log("Going to: " + xhttp2.getResponseHeader("Location"));
                                    window.location.href = xhttp2.getResponseHeader("Location");
                                }
                            } else {
                                document.getElementById("job_progress").innerHTML = this.responseText + "%";
                            }
                        } else {
                            document.getElementById("job_progress").innerHTML = "Error " + this.status;
                            clearInterval(poll_id);
                        }
                    }
                };
                xhttp2.open("GET", theLocation, true);
                xhttp2.send();
            } // function request()

        } else {
            document.getElementById("job_progress").innerHTML = "Error " + this.status;
        }
    }
}

function download_progress(job) {
    // var elem = document.getElementById("job_progress")
    // var width = 1;
    var xhttp1 = Ajax();
    xhttp1.onreadystatechange = update_progress
    xhttp1.open("GET", job, true);
    xhttp1.send(); 
}

function upload_progress(job, data) {
    var xhttp1 = Ajax();
    xhttp1.onreadystatechange = update_progress
    xhttp1.open("POST", job, true);
    xhttp1.send(data); 
}

function upload(obj, e) {
    e.preventDefault();
    if (!obj["firmware"].value) {
        alert("Missing file for upload!");
        return
    } 
    formData = new FormData();
    formData.append('firmware', obj["firmware"].files[0], obj["firmware"].files[0].name);
    $('#progress').show();
    upload_progress(obj.action, formData);
} 

function download(obj, e) {
    e.preventDefault();
    $('#progress').show();
    download_progress(obj.href); 
}
