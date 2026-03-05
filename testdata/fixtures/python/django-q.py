"""
Django Q objects for building complex query expressions.
"""

import operator
from functools import reduce


class Q:
    """
    Encapsulate filters as objects that can then be combined logically
    using `&` (AND) and `|` (OR) operators.
    """

    AND = "AND"
    OR = "OR"
    XOR = "XOR"
    conditional = True

    def __init__(self, *args, _connector=None, _negated=False, **kwargs):
        self.children = list(args) + sorted(kwargs.items())
        self.connector = _connector or self.AND
        self.negated = _negated

    def __and__(self, other):
        return self._combine(other, self.AND)

    def __or__(self, other):
        return self._combine(other, self.OR)

    def __xor__(self, other):
        return self._combine(other, self.XOR)

    def __invert__(self):
        obj = self.copy()
        obj.negated = not self.negated
        return obj

    def __repr__(self):
        template = "(NOT (%s: %s))" if self.negated else "(%s: %s)"
        return template % (self.connector, ", ".join(str(c) for c in self.children))

    def __bool__(self):
        return bool(self.children)

    def __len__(self):
        return len(self.children)

    def _combine(self, other, conn):
        """Combine this Q object with another using the given connector."""
        if not isinstance(other, Q):
            raise TypeError(other)

        if not self:
            return other.copy()
        if not other:
            return self.copy()

        obj = self.__class__()
        obj.connector = conn
        obj.add(self, conn)
        obj.add(other, conn)
        return obj

    def add(self, node, conn):
        """Add a node to the Q object, respecting connector types."""
        if node.connector == conn and not node.negated:
            self.children.extend(node.children)
        else:
            self.children.append(node)

    def copy(self):
        """Return a deep copy of this Q object."""
        obj = self.__class__(_connector=self.connector, _negated=self.negated)
        obj.children = self.children[:]
        return obj

    def resolve_expression(self, query, allow_joins=True, reuse=None, summarize=False):
        """Resolve the Q object into a WhereNode for the query compiler."""
        clause, joins = query._add_q(self, reuse, allow_joins=allow_joins, split_subq=False)
        query.promote_joins(joins)
        return clause

    @classmethod
    def create(cls, children, connector=AND, negated=False):
        """Create a new Q instance from a list of children."""
        obj = cls(_connector=connector, _negated=negated)
        obj.children = children
        return obj

    @classmethod
    def combine_queries(cls, queries, connector=AND):
        """Combine multiple Q objects with the given connector."""
        if not queries:
            return cls()
        return reduce(operator.and_ if connector == cls.AND else operator.or_, queries)
