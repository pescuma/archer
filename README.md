# archer
Get help understanding your architecture.

Please keep in mind that this is a work in progress, but it's already useful.

## Workspace
To use it, first you need to import the information into a workspace, than 
you can query and show graphs from it. 

A workspace can be set using the `-w` cli arg. Default is `./.archer` (if it exists) or `~/.archer`.

## Roots and projects

archer work with the idea of roots and projects. A project is a groping of source files or a 
table, for ex, depending on the importer.

A root is a groping of projects and allows to relate information from different sources.

## Importing information

### From gradle

Run
```
archer import gradle <root of your gradle project(s)>
```

All the projects will be imported. The root name is the name of the root gradle project.


### From hibernate

Run
```
archer import hibernate <source paths> --root <root name>
```

Imports hibernate configuration from files inside the source paths. Currently only
kotlin files are supported, and information is gathered from annotations. 

The `source paths` can be a path on disk or a query (see below).

The `root name` is required in this case and should be the schema name where the tables live.

You can also use `--glob` to filter which files should be parsed. For ex: `--glob '**/Db*.kt'`.


### From MySQL

Run
```
archer import mysql <connection string>
```

The `connection string` format is defined [here](https://github.com/go-sql-driver/mysql#dsn-data-source-name).

All tables will be imported. The root name is the schema name.


## Configuring things 

You can use `archer config set` to add information to the projects. 
Currently these are supported:
 - ignore: does not output this project
 - color: fixes the color to be used in graphs


## Showing data

### Textual output

Run
```
archer show
```

### Graphs

For this you need to have [graphviz dot](https://graphviz.org/) installed and in your path.

Run
```
archer graph -o <output file.extension>
```

### Selecting what to see

The simple version of the commands show all information available. This is usually too much, so
there are some parameters/filters to select what should be show.

#### `-r <name>`
Only show information from one root

### '-i <query>'
Only show information that matches the query (see below)

### '-e <query>'
Don't show information that matches the query (see below)

## Queries

Queries allows you to select which projects are interesting. The supported formats are:

### Project query: `<root name>:<project name>` or `<project name>` or `<project simple name>`
You can use `*` inside there to avoid some typing.

### Any dependency query: `<project 1 query> -> <project 2query>`
Shows any dependency chain that links project 1 to project 2

### Max steps dependency query: `<project 1 query> -N-> <project 2query>`
`N` is a number.

Shows any dependency chain that links project 1 to project 2, with at most N hops.

### Invert query: `not:<query>`
Inverts the matching.

### Multiple matches required: `<query> & <query>`
To do an OR just pass different `-i` arguments.







