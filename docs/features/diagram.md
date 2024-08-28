# Diagram

MangoSQL comes with the ability to generate ERD Diagram from your database schema.
This is really convenient to visualize your schema in documentation or CI.

```sh
mangosql diagram ./schema.sql
```

* `./schema.sql`: Input schema, accept a SQL file or a directory of migrations
* `--output`: Output where the svg file generated will be written.
* `--title`: Title of the diagram
* `--meta`: Metadata of the diagram, separated by `|`
* `--sketch`: Enable Sketch mode
* `--dark`: Enable Dark mode

## Demo

### Standard

```sh
mangosql diagram ./schema.sql
```

![](/blog.svg)

### Sketch without info

```sh
mangosql diagram -t "" ./schema.sql
```

![](/blog_simple.svg)

### Dark Mode with custom info

```sh
mangosql diagram --sketch --dark --title "My wonderful Blog" --meta "Version: 1.0.2" ./schema.sql
```

![](/blog_dark.svg)