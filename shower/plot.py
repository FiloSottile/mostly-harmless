import numpy as np
import matplotlib.pyplot as plt

data = [
    ("red wine", 19, 32),
    ("can of soda", 19, 32),
    ("mangoes", 17, 35),
    ("oranges", 10, 41),
    ("popsicle", 18, 33),
    ("mashed potatoes", 10, 41),
    ("water", 30, 21),
    ("soup", 11, 40),
    ("sandwich", 14, 38),
    ("candy bar", 9, 41),
    ("smoke", 9, 43),
    ("beer", 20, 31),
    ("ice coffee", 17, 34),
    ("hot latte", 17, 32),
    ("grapes", 15, 35),
]

score = lambda x: x[1] / (x[1] + x[2]) * 50
data.sort(key=score)

with plt.xkcd():
    plt.rc("font", family="xkcd")
    ind = np.arange(len(data))  # the x locations for the groups
    width = 0.45  # the width of the bars: can also be len(x) sequence
    fig = plt.figure(figsize=(9, 9))
    ax = plt.subplot(111)
    ax.spines["right"].set_visible(False)
    ax.spines["top"].set_visible(False)
    p1 = plt.bar(ind, [score(p) for p in data], width, color="black")
    plt.title("Is it ok to eat it in the shower?".upper())
    plt.xticks(ind, [p[0].upper() for p in data])
    plt.yticks(np.arange(0, 51, 10))
    fig.autofmt_xdate()
    plt.tight_layout()

plt.savefig("shower.png", dpi=300)
plt.show()
