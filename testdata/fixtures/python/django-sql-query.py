"""
Django-style SQL query compiler infrastructure.
"""


class SQLQuery:
    """Represent a structured SQL query."""

    compiler = "SQLCompiler"
    _db = None

    def __init__(self, model=None, where=None, alias_map=None):
        self.model = model
        self.where = where or WhereNode()
        self.alias_map = alias_map or {}
        self.tables = []
        self.select = []
        self.group_by = None
        self.order_by = []
        self.distinct = False
        self.distinct_fields = []
        self.standard_ordering = True
        self.select_related = False
        self.max_depth = 5
        self.values_select = ()
        self.annotation_select_mask = None
        self.extra_select_mask = None
        self._extra = {}
        self._annotations = {}

    def as_sql(self, compiler=None, connection=None):
        """Return the query as an SQL string and parameters."""
        compiler = compiler or self.get_compiler(connection=connection)
        return compiler.as_sql()

    def get_compiler(self, using=None, connection=None, elide_empty=True):
        """Return a compiler instance for this query."""
        if using is None and connection is None:
            raise ValueError("Need either using or connection")
        if using:
            from django.db import connections
            connection = connections[using]
        return connection.ops.compiler(self.compiler)(
            self, connection, using, elide_empty=elide_empty
        )

    def add_filter(self, filter_lhs, filter_rhs, connector="AND", negate=False):
        """Add a single filter to the query."""
        clause = self._build_lookup(filter_lhs, filter_rhs)
        if negate:
            clause.negate()
        self.where.add(clause, connector)

    def add_ordering(self, *ordering):
        """Add items from the 'ordering' sequence to the query's order by."""
        errors = []
        for item in ordering:
            if isinstance(item, str) and item == "?":
                continue
            if hasattr(item, "resolve_expression"):
                continue
            if isinstance(item, str):
                if item.startswith("-"):
                    item = item[1:]
                if item in ("pk",):
                    continue
                if not self._check_field_name(item):
                    errors.append(item)
        if errors:
            raise FieldError("Invalid order_by arguments: %s" % errors)
        self.order_by.extend(ordering)

    def clone(self):
        """Return a copy of the current query."""
        obj = self.__class__(model=self.model)
        obj.where = self.where.clone()
        obj.alias_map = self.alias_map.copy()
        obj.tables = self.tables[:]
        obj.select = self.select[:]
        obj.group_by = self.group_by
        obj.order_by = self.order_by[:]
        obj.distinct = self.distinct
        obj.distinct_fields = self.distinct_fields[:]
        obj.standard_ordering = self.standard_ordering
        obj.select_related = self.select_related
        obj.max_depth = self.max_depth
        obj.values_select = self.values_select
        obj._extra = self._extra.copy()
        obj._annotations = self._annotations.copy()
        return obj

    def _build_lookup(self, lhs, rhs):
        """Build a WhereNode from a filter expression."""
        return WhereNode(children=[(lhs, rhs)])

    def _check_field_name(self, name):
        """Check if a field name is valid for the model."""
        if self.model is None:
            return True
        return hasattr(self.model, name) or name in self.annotation_select_mask or {}


class WhereNode:
    """Represent an SQL WHERE clause."""

    default_connector = "AND"

    def __init__(self, children=None, connector=None, negated=False):
        self.children = children or []
        self.connector = connector or self.default_connector
        self.negated = negated

    def add(self, node, connector):
        if self.connector == connector:
            self.children.append(node)
        else:
            new_node = WhereNode(children=self.children[:], connector=self.connector)
            self.children = [new_node, node]
            self.connector = connector

    def negate(self):
        self.negated = not self.negated

    def clone(self):
        obj = WhereNode(
            children=[c.clone() if hasattr(c, "clone") else c for c in self.children],
            connector=self.connector,
            negated=self.negated,
        )
        return obj


class FieldError(Exception):
    """Raised when a field lookup encounters a problem."""
    pass
