
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
