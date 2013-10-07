---
layout: post
title: "Instance monkey-patching in Python"
date: 2013-10-07 00:19
comments: true
categories: python
---

[Monkey-patching][monkey] is the technique of swapping functions or methods with others in order to change a module, library or class behavior.

There are some people with strong opinions about it. I haven't, but it comes really useful when testing, to simulate side-effecting functions or to silence expected errors and warnings.

**Class methods** monkey patching in Python is really easy, as you can freely assign function to class method names:

```python
>>> class Class():
...    def add(self, x, y):
...       return x + y
...
>>> inst = Class()
>>> def not_exactly_add(self, x, y):
...    return x * y
...
>>> Class.add = not_exactly_add
>>> inst.add(3, 3)
9
```

<!-- more -->

This way **all the instances** of the target class will have the method monkey-patched and there is no problem with arguments, bindings... Everything really straight-forward.

We can also call the old existing method, to handle only some cases or to add some functionality while not repeating code ([DRY][dry]):

```python
>>> class Class():
...    def add(self, x, y):
...       return x + y
...
>>> old_boring_add = Class.add
>>> def add_is_not_enough(self, x, y):
...    return old_boring_add(self, x, y) + 1
...
>>> inst = Class()
>>> inst.add(3, 3)
6
>>> Class.add = add_is_not_enough
>>> inst.add(3, 3)
7
```

<!-- Finally, we might want to monkey-patch repeatedly, maybe dinamically, and so have each monkey-patch to **build on top of the previous one**. Easy done,  -->

But what if we wanted to do the same, patching **just a single instance**?

To recap, the requirements are:

* we want just the current instance to be patched;
* we want to build something on top of the existing method, not to replace it entirely;
* we want each monkey-patch not to rollback all the previous ones (so no [`super()`][super] or class method call);
* we want to be able to do so also from inside a method.

The trick is to save and use the existing method as we did above, and then **bind the new function to the instance** with [`types.MethodType`][methodtype] before assigning it to the method name.

The binding is the magic that causes the instance to be passed as first argument (`self`) each time the method is called. See [these][stack1] [two][stack2] StackOverflow questions to get an idea.

```python
>>> import types
>>> class Class():
...    def add(self, x, y):
...       return x + y
...    def become_more_powerful(self):
...       old_add = self.add
...       def more_powerful_add(self, x, y):
...          return old_add(x, y) + 1
...       self.add = types.MethodType(more_powerful_add, self)
...
>>> inst = Class()
>>> inst.add(3, 3)
6
>>> inst.become_more_powerful()
>>> inst.add(3, 3)
7
>>> inst.become_more_powerful()
>>> inst.become_more_powerful()
>>> inst.become_more_powerful()
>>> inst.add(3, 3)
10
```

And here we go!

## A practical example

You can see this technique being used in [*youtube-dl*][ytdl] to silence expected warnings in [this commit][commit].

The monkey-patching of the instance is done on itself by a method of a testing subclass of the downloader.


[super]: http://docs.python.org/2/library/functions.html#super
[dry]: https://en.wikipedia.org/wiki/Don't_repeat_yourself
[monkey]: https://en.wikipedia.org/wiki/Monkey_patch
[stack1]: http://stackoverflow.com/questions/114214/class-method-differences-in-python-bound-unbound-and-static
[stack2]: http://stackoverflow.com/questions/136097/what-is-the-difference-between-staticmethod-and-classmethod-in-python
[ytdl]: https://github.com/rg3/youtube-dl
[commit]: https://github.com/rg3/youtube-dl/commit/00fcc17aeeab11ce694699bf183d33a3af75aab6
[methodtype]: http://docs.python.org/2/library/types.html#types.MethodType
