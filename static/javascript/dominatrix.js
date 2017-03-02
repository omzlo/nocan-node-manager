/*
 * See https://medium.freecodecamp.com/how-to-write-a-jquery-like-library-in-71-lines-of-code-learn-about-the-dom-e9fb99dbc8d2#.t7d6m6wsd
 * See http://youmightnotneedjquery.com
 */

var Dominatrix = function(selector) {
     this.selector = selector || null;
     this.element = null;
};

Dominatrix.prototype.init = function() {
    switch (this.selector[0]) {
        case '<':
            var matches = this.selector.match(/<([\w-]*)>/);
            if (matches === null || matches === undefined) {
                throw 'Invalid Selector / Node';
                return false;
            }
            var nodeName = matches[0].replace('<', '').replace('>', '');
            this.element = document.createElement(nodeName);
            break;
        default:
            this.element = document.querySelector(this.selector);
    }
};

Dominatrix.prototype.on = function(event, callback) {
     var evt = this.eventHandler.bindEvent(event, callback, this.element);
}

Dominatrix.prototype.off = function(event) {
     var evt = this.eventHandler.unbindEvent(event, this.element);
}

Dominatrix.prototype.val = function(newVal) {
     return (newVal !== undefined ? this.element.value = newVal : this.element.value);
};

Dominatrix.prototype.append = function(html) {
     this.element.innerHTML = this.element.innerHTML + html;
};

Dominatrix.prototype.prepend = function(html) {
     this.element.innerHTML = html + this.element.innerHTML;
};

Dominatrix.prototype.html = function(html) {
    if (html === undefined) {
        return this.element.innerHTML;
    }
    this.element.innerHTML = html;
};

Dominatrix.prototype.show = function() {
    this.element.style.display = 'block';
};

Dominatrix.prototype.hide = function() {
    this.element.style.display = 'none';
};

Dominatrix.prototype.eventHandler = {
    events: [],
    bindEvent: function(event, callback, targetElement) {
        this.unbindEvent(event, targetElement);
        targetElement.addEventListener(event, callback, false);
        this.events.push({
            type: event,
            event: callback,
            target: targetElement
        });
    },
    findEvent: function(event) {
                   return this.events.filter(function(evt) {
                       return (evt.type === event);
                   }, event)[0];
               },
    unbindEvent: function(event, targetElement) {
                     var foundEvent = this.findEvent(event);
                     if (foundEvent !== undefined) {
                         targetElement.removeEventListener(event, foundEvent.event, false);
                     }
                     this.events = this.events.filter(function(evt) {
                         return (evt.type !== event);
                     }, event);
                 }
};

$ = function(selector) {
     var el = new Dominatrix(selector);
     el.init();
     return el;
}

Ajax = function() {
    var xhttp;
    if (window.XMLHttpRequest) {
            xhttp = new XMLHttpRequest();
    } else {
        // code for IE6, IE5
        xhttp = new ActiveXObject("Microsoft.XMLHTTP");
    }
    return xhttp;
}
