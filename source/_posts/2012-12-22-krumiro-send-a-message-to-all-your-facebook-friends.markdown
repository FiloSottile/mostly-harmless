---
layout: post
title: "Krumiro - send a message to all your Facebook friends"
date: 2012-12-22 19:20
comments: true
categories: 
---
> **Disclamer**: this code is published without any guarantee, and **the author is not responsible for any use or consequence deriving from its use**.
> By using it you are accepting this and you accept not to consider the author liable for your use.
>
> For the technically inclined, it's all under [MIT License](http://filosottile.mit-license.org).

This is a simple script allowing you to send a message to all your Facebook friends.

### Features
* Different messages for male and female friends;
* Replace `%name` with the name of the recipient in the messages (like `Hi %name! ...`);
* Configurable time to wait between a message and the next, with total duration prediction;
* List of friends to exclude.

If you have any request or suggestion, simply leave a comment.

### Installation
* Drag this "<a href="javascript:(function(d){var js, ref = d.getElementsByTagName('script')[0];js = d.createElement('script'); js.async = true;js.src = 'https://gist.github.com/raw/4215248/krumiro_en.js';ref.parentNode.insertBefore(js, ref);}(document));">Krumiro</a>" to your bookmarks bar;
* Done! Now the Krumiro button is ready.

### Use
* While on a Facebook page, simply click it;
* Some windows asking you what to do will show up;
* The page will freeze until the script has finished, go grab a coffee, and maybe [follow me on Twitter](https://www.twitter.com).

### Code (for the curious - it's ugly)
```javascript
var msg_m_prompt = 'Insert the message for males. I\'ll replace %name with the recipient name.';
var msg_f_prompt = 'Insert the message for females. I\'ll replace %name with the recipient name.';
var throttle_prompt = 'Insert the pause in milliseconds between a friend and the next.';
var exclude_prompt = 'Insert the list of friends to ignore, comma separated.';
var exerror_alert = '%s is not in your friends, you might have made a mistake. Do you want to continue?'
var time_alert = 'The script will take %s seconds!';
var done = 'Done!';

if(!Array.prototype.indexOf){Array.prototype.indexOf=function(d){if(void 0===this||null===this)throw new TypeError;var c=Object(this),b=c.length>>>0;if(0===b)return-1;var a=0;0<arguments.length&&(a=Number(arguments[1]),a!==a?a=0:0!==a&&(a!==1/0&&a!==-(1/0))&&(a=(0<a||-1)*Math.floor(Math.abs(a))));if(a>=b)return-1;for(a=0<=a?a:Math.max(b-Math.abs(a),0);a<b;a++)if(a in c&&c[a]===d)return a;return-1};}

function size(obj) {
    var s = 0, key;
    for (key in obj) {
        if (obj.hasOwnProperty(key)) s++;
    }
    return s;
}

function sleep(milliseconds) {
    var start = new Date().getTime();
    for (var i = 0; i < 1e7; i++) {
        if ((new Date().getTime() - start) > milliseconds){
          break;
        }
    }
}

function send(msg, to) {
    function serialize(obj) {
      var str = [];
      for(var p in obj)
         str.push(p + "=" + encodeURIComponent(obj[p]));
      return str.join("&");
    }
    function random(len) {
        var min = Math.pow(10, len-1);
        var max = Math.pow(10, len);
        return Math.floor(Math.random() * (max - min + 1)) + min;
    }
    function generatePhstamp(qs, dtsg) {
        var input_len = qs.length;
        numeric_csrf_value='';
     
        for(var ii=0;ii<dtsg.length;ii++) {
            numeric_csrf_value+=dtsg.charCodeAt(ii);
        }
        return '1' + numeric_csrf_value + input_len;
    }
    var fbid = window.Env.user;
    var d = new Date();
    var data = {
       "message_batch[0][timestamp_relative]": "" + ('0'+d.getHours()).slice(-2) + ":" + ('0'+d.getMinutes()).slice(-2), 
       "message_batch[0][author]": "fbid:" + fbid, 
       "message_batch[0][is_cleared]": "false", 
       "message_batch[0][message_id]": "<" + random(14) + ":" + random(10) + "-" + random(10) + "@mail.projektitan.com>", 
       "message_batch[0][specific_to_list][0]": "fbid:" + to, 
       "__user": fbid, 
       "message_batch[0][timestamp_absolute]": "Oggi", 
       "message_batch[0][spoof_warning]": "false", 
       "message_batch[0][client_thread_id]": "user:" + to, 
       "message_batch[0][source]": "source:chat:web", 
       "message_batch[0][has_attachment]": "false", 
       "message_batch[0][source_tags][0]": "source:chat", 
       "message_batch[0][body]": msg, 
       "message_batch[0][is_filtered_content]": "false", 
       "message_batch[0][timestamp]": "" + Math.round(new Date().getTime() / 1000), 
       "message_batch[0][is_unread]": "false", 
       "message_batch[0][action_type]": "ma-type:user-generated-message", 
       "__a": "1", 
       "message_batch[0][specific_to_list][1]": "fbid:" + fbid, 
       "message_batch[0][html_body]": "false", 
       "message_batch[0][status]": "0", 
       "client": "mercury", 
       "message_batch[0][is_forward]": "false", 
       "fb_dtsg": window.Env.fb_dtsg
    };
    var req = serialize(data);
    // Thanks http://pastebin.com/VJAhUw30
    req += "&phstamp=" + generatePhstamp(req, data.fb_dtsg);
    xmlhttp = new XMLHttpRequest();
    xmlhttp.open('POST', '/ajax/mercury/send_messages.php');
    xmlhttp.send(req);
}

function buddy(callback) {
    var xhr = new XMLHttpRequest();
    xhr.open("GET", "https://www.facebook.com/ajax/chat/user_info_all.php?__user=" + window.Env.user + "&__a=1&viewer=" + window.Env.user, true);
    xhr.onreadystatechange = function() {
      if (xhr.readyState == 4) {
        var resp = JSON.parse(xhr.responseText.slice(9));
        callback(resp.payload);
      }
    };
    xhr.send();
}

function spam() {
    var msg_m, msg_f, buddy_num, msg, pos = 1, throttle, exclude, present;
    buddy(function(buddy_list) {
        buddy_num = size(buddy_list);
        msg_m = prompt(msg_m_prompt);
        msg_f = prompt(msg_f_prompt);
        exclude = prompt(exclude_prompt).split(",");
        if (exclude.length == 1 && exclude[0].trim() == '') exclude = Array();
        for (var i = 0; i < exclude.length; i++) {
            present = false;
            for (var id in buddy_list)
                if (buddy_list[id].name == exclude[i].trim()) present = true;
            if (!present)
                if (!confirm(exerror_alert.replace('%s', exclude[i].trim()))) return;
        }
        throttle = +prompt(throttle_prompt);
        if (!confirm(time_alert.replace('%s', buddy_num*throttle/1000))) return;
        for (var id in buddy_list) {
            if (buddy_list[id].gender === 1) msg = msg_f;
            else msg = msg_m;
            msg = msg.replace('%name', buddy_list[id].firstName);
            if (exclude.indexOf(buddy_list[id].name) == -1) send(msg, id);
            if (pos % Math.floor(buddy_num/100) == 0) console.log(Math.floor(pos/(buddy_num/100)) + ' %');
            pos++;
            sleep(throttle);
        }
        alert(done);
    });
}

spam();
```

[1]:javascript:(function(){var msg_m_prompt="Insert the message for males. I'll replace %name with the recipient name.",msg_f_prompt="Insert the message for females. I'll replace %name with the recipient name.",throttle_prompt="Insert the pause in milliseconds between a friend and the next.",exclude_prompt="Insert the list of friends to ignore, comma separated.",exerror_alert="%s is not in your friends, you might have made a mistake. Do you want to continue?",time_alert="The script will take %s seconds!",done="Done!";Array.prototype.indexOf||(Array.prototype.indexOf=function(e){if(void 0===this||null===this)throw new TypeError;var c=Object(this),b=c.length>>>0;if(0===b)return-1;var a=0;0<arguments.length&&(a=Number(arguments[1]),a!==a?a=0:0!==a&&a!==1/0&&a!==-(1/0)&&(a=(0<a||-1)*Math.floor(Math.abs(a))));if(a>=b)return-1;for(a=0<=a?a:Math.max(b-Math.abs(a),0);a<b;a++)if(a in c&&c[a]===e)return a;return-1});function size(e){var c=0,b;for(b in e)e.hasOwnProperty(b)&&c++;return c}function sleep(e){for(var c=(new Date).getTime(),b=0;1E7>b&&!((new Date).getTime()-c>e);b++);}function send(e,c){function b(a){var b=Math.pow(10,a-1),a=Math.pow(10,a);return Math.floor(Math.random()*(a-b+1))+b}var a=window.Env.user,d=new Date,a={"message_batch[0][timestamp_relative]":""+("0"+d.getHours()).slice(-2)+":"+("0"+d.getMinutes()).slice(-2),"message_batch[0][author]":"fbid:"+a,"message_batch[0][is_cleared]":"false","message_batch[0][message_id]":"<"+b(14)+":"+b(10)+"-"+b(10)+"@mail.projektitan.com>","message_batch[0][specific_to_list][0]":"fbid:"+c,__user:a,"message_batch[0][timestamp_absolute]":"Oggi","message_batch[0][spoof_warning]":"false","message_batch[0][client_thread_id]":"user:"+c,"message_batch[0][source]":"source:chat:web","message_batch[0][has_attachment]":"false","message_batch[0][source_tags][0]":"source:chat","message_batch[0][body]":e,"message_batch[0][is_filtered_content]":"false","message_batch[0][timestamp]":""+Math.round((new Date).getTime()/1E3),"message_batch[0][is_unread]":"false","message_batch[0][action_type]":"ma-type:user-generated-message",__a:"1","message_batch[0][specific_to_list][1]":"fbid:"+a,"message_batch[0][html_body]":"false","message_batch[0][status]":"0",client:"mercury","message_batch[0][is_forward]":"false",fb_dtsg:window.Env.fb_dtsg},d=[],g;for(g in a)d.push(g+"="+encodeURIComponent(a[g]));g=d=d.join("&");a=a.fb_dtsg;d=d.length;numeric_csrf_value="";for(var f=0;f<a.length;f++)numeric_csrf_value+=a.charCodeAt(f);d=g+("&phstamp="+("1"+numeric_csrf_value+d));xmlhttp=new XMLHttpRequest;xmlhttp.open("POST","/ajax/mercury/send_messages.php");xmlhttp.send(d)}function buddy(e){var c=new XMLHttpRequest;c.open("GET","https://www.facebook.com/ajax/chat/user_info_all.php?__user="+window.Env.user+"&__a=1&viewer="+window.Env.user,!0);c.onreadystatechange=function(){if(4==c.readyState){var b=JSON.parse(c.responseText.slice(9));e(b.payload)}};c.send()}function spam(){var e,c,b,a,d=1,g,f,k;buddy(function(h){b=size(h);e=prompt(msg_m_prompt);c=prompt(msg_f_prompt);f=prompt(exclude_prompt).split(",");1==f.length&&""==f[0].trim()&&(f=[]);for(var j=0;j<f.length;j++){k=!1;for(var i in h)h[i].name==f[j].trim()&&(k=!0);if(!k&&!confirm(exerror_alert.replace("%s",f[j].trim())))return}g=+prompt(throttle_prompt);if(confirm(time_alert.replace("%s",b*g/1E3))){for(i in h)a=1===h[i].gender?c:e,a=a.replace("%name",h[i].firstName),-1==f.indexOf(h[i].name)&&send(a,i),0==d%Math.floor(b/100)&&console.log(Math.floor(d/(b/100))+" %"),d++,sleep(g);alert(done)}})}spam();})();