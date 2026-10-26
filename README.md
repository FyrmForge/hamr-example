# hamr-example

Example apps built with [HAMR](https://github.com/FyrmForge/hamr). Each app lives in its own folder with its own Go module.

| Folder | What it is | Guide |
|---|---|---|
| [`gather/`](gather/) | Gather, an events app: sign up, create events, RSVP, email the organiser, buy a ticket through the Stripe mock. | [Build-along guide](https://hamrstack.io/docs/guide), 12 steps |

## Following a guide step by step

Every step of a guide is a tag named `<app>/step-NN`, for example `gather/step-05`.

```bash
git clone https://github.com/FyrmForge/hamr-example && cd hamr-example
git checkout gather/step-04   # the code at the start of step 5
cd gather && hamr dev
```

Compare two steps with `git diff gather/step-04 gather/step-05`, or on GitHub at
`compare/gather/step-04...gather/step-05`.

