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
<!-- more -->
### Code (for the curious - it's ugly)
{% gist 4215248 krumiro_en.js %}

[1]:javascript:(function(){var msg_m_prompt="Insert the message for males. I'll replace %name with the recipient name.",msg_f_prompt="Insert the message for females. I'll replace %name with the recipient name.",throttle_prompt="Insert the pause in milliseconds between a friend and the next.",exclude_prompt="Insert the list of friends to ignore, comma separated.",exerror_alert="%s is not in your friends, you might have made a mistake. Do you want to continue?",time_alert="The script will take %s seconds!",done="Done!";Array.prototype.indexOf||(Array.prototype.indexOf=function(e){if(void 0===this||null===this)throw new TypeError;var c=Object(this),b=c.length>>>0;if(0===b)return-1;var a=0;0<arguments.length&&(a=Number(arguments[1]),a!==a?a=0:0!==a&&a!==1/0&&a!==-(1/0)&&(a=(0<a||-1)*Math.floor(Math.abs(a))));if(a>=b)return-1;for(a=0<=a?a:Math.max(b-Math.abs(a),0);a<b;a++)if(a in c&&c[a]===e)return a;return-1});function size(e){var c=0,b;for(b in e)e.hasOwnProperty(b)&&c++;return c}function sleep(e){for(var c=(new Date).getTime(),b=0;1E7>b&&!((new Date).getTime()-c>e);b++);}function send(e,c){function b(a){var b=Math.pow(10,a-1),a=Math.pow(10,a);return Math.floor(Math.random()*(a-b+1))+b}var a=window.Env.user,d=new Date,a={"message_batch[0][timestamp_relative]":""+("0"+d.getHours()).slice(-2)+":"+("0"+d.getMinutes()).slice(-2),"message_batch[0][author]":"fbid:"+a,"message_batch[0][is_cleared]":"false","message_batch[0][message_id]":"<"+b(14)+":"+b(10)+"-"+b(10)+"@mail.projektitan.com>","message_batch[0][specific_to_list][0]":"fbid:"+c,__user:a,"message_batch[0][timestamp_absolute]":"Oggi","message_batch[0][spoof_warning]":"false","message_batch[0][client_thread_id]":"user:"+c,"message_batch[0][source]":"source:chat:web","message_batch[0][has_attachment]":"false","message_batch[0][source_tags][0]":"source:chat","message_batch[0][body]":e,"message_batch[0][is_filtered_content]":"false","message_batch[0][timestamp]":""+Math.round((new Date).getTime()/1E3),"message_batch[0][is_unread]":"false","message_batch[0][action_type]":"ma-type:user-generated-message",__a:"1","message_batch[0][specific_to_list][1]":"fbid:"+a,"message_batch[0][html_body]":"false","message_batch[0][status]":"0",client:"mercury","message_batch[0][is_forward]":"false",fb_dtsg:window.Env.fb_dtsg},d=[],g;for(g in a)d.push(g+"="+encodeURIComponent(a[g]));g=d=d.join("&");a=a.fb_dtsg;d=d.length;numeric_csrf_value="";for(var f=0;f<a.length;f++)numeric_csrf_value+=a.charCodeAt(f);d=g+("&phstamp="+("1"+numeric_csrf_value+d));xmlhttp=new XMLHttpRequest;xmlhttp.open("POST","/ajax/mercury/send_messages.php");xmlhttp.send(d)}function buddy(e){var c=new XMLHttpRequest;c.open("GET","https://www.facebook.com/ajax/chat/user_info_all.php?__user="+window.Env.user+"&__a=1&viewer="+window.Env.user,!0);c.onreadystatechange=function(){if(4==c.readyState){var b=JSON.parse(c.responseText.slice(9));e(b.payload)}};c.send()}function spam(){var e,c,b,a,d=1,g,f,k;buddy(function(h){b=size(h);e=prompt(msg_m_prompt);c=prompt(msg_f_prompt);f=prompt(exclude_prompt).split(",");1==f.length&&""==f[0].trim()&&(f=[]);for(var j=0;j<f.length;j++){k=!1;for(var i in h)h[i].name==f[j].trim()&&(k=!0);if(!k&&!confirm(exerror_alert.replace("%s",f[j].trim())))return}g=+prompt(throttle_prompt);if(confirm(time_alert.replace("%s",b*g/1E3))){for(i in h)a=1===h[i].gender?c:e,a=a.replace("%name",h[i].firstName),-1==f.indexOf(h[i].name)&&send(a,i),0==d%Math.floor(b/100)&&console.log(Math.floor(d/(b/100))+" %"),d++,sleep(g);alert(done)}})}spam();})();
