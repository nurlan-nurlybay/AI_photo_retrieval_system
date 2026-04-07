from pydantic import BaseModel, Field, field_validator
from typing import List, Optional
from typing import Annotated
from typing import Callable
from typing import ClassVar

MutantDict = Annotated[dict[str, Callable], "Mutant"] # type: ignore


def _mutmut_trampoline(orig, mutants, call_args, call_kwargs, self_arg = None): # type: ignore
    """Forward call to original or mutated function, depending on the environment"""
    import os # type: ignore
    mutant_under_test = os.environ['MUTANT_UNDER_TEST'] # type: ignore
    if mutant_under_test == 'fail': # type: ignore
        from mutmut.__main__ import MutmutProgrammaticFailException # type: ignore
        raise MutmutProgrammaticFailException('Failed programmatically')       # type: ignore
    elif mutant_under_test == 'stats': # type: ignore
        from mutmut.__main__ import record_trampoline_hit # type: ignore
        record_trampoline_hit(orig.__module__ + '.' + orig.__name__) # type: ignore
        # (for class methods, orig is bound and thus does not need the explicit self argument)
        result = orig(*call_args, **call_kwargs) # type: ignore
        return result # type: ignore
    prefix = orig.__module__ + '.' + orig.__name__ + '__mutmut_' # type: ignore
    if not mutant_under_test.startswith(prefix): # type: ignore
        result = orig(*call_args, **call_kwargs) # type: ignore
        return result # type: ignore
    mutant_name = mutant_under_test.rpartition('.')[-1] # type: ignore
    if self_arg is not None: # type: ignore
        # call to a class method where self is not bound
        result = mutants[mutant_name](self_arg, *call_args, **call_kwargs) # type: ignore
    else:
        result = mutants[mutant_name](*call_args, **call_kwargs) # type: ignore
    return result # type: ignore

def validate_namespace(v: str) -> str:
    args = [v]# type: ignore
    kwargs = {}# type: ignore
    return _mutmut_trampoline(x_validate_namespace__mutmut_orig, x_validate_namespace__mutmut_mutants, args, kwargs, None)

def x_validate_namespace__mutmut_orig(v: str) -> str:
    v = v.strip()
    if not v or v.lower() in ["default", "none", "null"]:
        raise ValueError("A specific user namespace is required. 'default' is not allowed.")
    return v

def x_validate_namespace__mutmut_1(v: str) -> str:
    v = None
    if not v or v.lower() in ["default", "none", "null"]:
        raise ValueError("A specific user namespace is required. 'default' is not allowed.")
    return v

def x_validate_namespace__mutmut_2(v: str) -> str:
    v = v.strip()
    if not v and v.lower() in ["default", "none", "null"]:
        raise ValueError("A specific user namespace is required. 'default' is not allowed.")
    return v

def x_validate_namespace__mutmut_3(v: str) -> str:
    v = v.strip()
    if v or v.lower() in ["default", "none", "null"]:
        raise ValueError("A specific user namespace is required. 'default' is not allowed.")
    return v

def x_validate_namespace__mutmut_4(v: str) -> str:
    v = v.strip()
    if not v or v.upper() in ["default", "none", "null"]:
        raise ValueError("A specific user namespace is required. 'default' is not allowed.")
    return v

def x_validate_namespace__mutmut_5(v: str) -> str:
    v = v.strip()
    if not v or v.lower() not in ["default", "none", "null"]:
        raise ValueError("A specific user namespace is required. 'default' is not allowed.")
    return v

def x_validate_namespace__mutmut_6(v: str) -> str:
    v = v.strip()
    if not v or v.lower() in ["XXdefaultXX", "none", "null"]:
        raise ValueError("A specific user namespace is required. 'default' is not allowed.")
    return v

def x_validate_namespace__mutmut_7(v: str) -> str:
    v = v.strip()
    if not v or v.lower() in ["DEFAULT", "none", "null"]:
        raise ValueError("A specific user namespace is required. 'default' is not allowed.")
    return v

def x_validate_namespace__mutmut_8(v: str) -> str:
    v = v.strip()
    if not v or v.lower() in ["default", "XXnoneXX", "null"]:
        raise ValueError("A specific user namespace is required. 'default' is not allowed.")
    return v

def x_validate_namespace__mutmut_9(v: str) -> str:
    v = v.strip()
    if not v or v.lower() in ["default", "NONE", "null"]:
        raise ValueError("A specific user namespace is required. 'default' is not allowed.")
    return v

def x_validate_namespace__mutmut_10(v: str) -> str:
    v = v.strip()
    if not v or v.lower() in ["default", "none", "XXnullXX"]:
        raise ValueError("A specific user namespace is required. 'default' is not allowed.")
    return v

def x_validate_namespace__mutmut_11(v: str) -> str:
    v = v.strip()
    if not v or v.lower() in ["default", "none", "NULL"]:
        raise ValueError("A specific user namespace is required. 'default' is not allowed.")
    return v

def x_validate_namespace__mutmut_12(v: str) -> str:
    v = v.strip()
    if not v or v.lower() in ["default", "none", "null"]:
        raise ValueError(None)
    return v

def x_validate_namespace__mutmut_13(v: str) -> str:
    v = v.strip()
    if not v or v.lower() in ["default", "none", "null"]:
        raise ValueError("XXA specific user namespace is required. 'default' is not allowed.XX")
    return v

def x_validate_namespace__mutmut_14(v: str) -> str:
    v = v.strip()
    if not v or v.lower() in ["default", "none", "null"]:
        raise ValueError("a specific user namespace is required. 'default' is not allowed.")
    return v

def x_validate_namespace__mutmut_15(v: str) -> str:
    v = v.strip()
    if not v or v.lower() in ["default", "none", "null"]:
        raise ValueError("A SPECIFIC USER NAMESPACE IS REQUIRED. 'DEFAULT' IS NOT ALLOWED.")
    return v

x_validate_namespace__mutmut_mutants : ClassVar[MutantDict] = { # type: ignore
'x_validate_namespace__mutmut_1': x_validate_namespace__mutmut_1, 
    'x_validate_namespace__mutmut_2': x_validate_namespace__mutmut_2, 
    'x_validate_namespace__mutmut_3': x_validate_namespace__mutmut_3, 
    'x_validate_namespace__mutmut_4': x_validate_namespace__mutmut_4, 
    'x_validate_namespace__mutmut_5': x_validate_namespace__mutmut_5, 
    'x_validate_namespace__mutmut_6': x_validate_namespace__mutmut_6, 
    'x_validate_namespace__mutmut_7': x_validate_namespace__mutmut_7, 
    'x_validate_namespace__mutmut_8': x_validate_namespace__mutmut_8, 
    'x_validate_namespace__mutmut_9': x_validate_namespace__mutmut_9, 
    'x_validate_namespace__mutmut_10': x_validate_namespace__mutmut_10, 
    'x_validate_namespace__mutmut_11': x_validate_namespace__mutmut_11, 
    'x_validate_namespace__mutmut_12': x_validate_namespace__mutmut_12, 
    'x_validate_namespace__mutmut_13': x_validate_namespace__mutmut_13, 
    'x_validate_namespace__mutmut_14': x_validate_namespace__mutmut_14, 
    'x_validate_namespace__mutmut_15': x_validate_namespace__mutmut_15
}
x_validate_namespace__mutmut_orig.__name__ = 'x_validate_namespace'

# ---------- Base ----------
class NamespaceRequest(BaseModel):
    namespace: str = Field(..., min_length=1, description="Unique user namespace (collection).")
    
    @field_validator('namespace')
    @classmethod
    def check_namespace(cls, v: str) -> str:
        return validate_namespace(v)

# ---------- Add ----------
class VectorAddRequest(NamespaceRequest):
    id: int = Field(..., description="Media ID")
    vector: List[float]
    normalize: bool = True

class VectorAddResponse(BaseModel):
    ok: bool
    id: int
    namespace: str
    dim: Optional[int] = None
    error: Optional[str] = None

# ---------- Delete ----------
class VectorDeleteRequest(NamespaceRequest):
    id: int

class VectorDeleteResponse(BaseModel):
    ok: bool
    id: int
    namespace: str
    deleted: bool
    error: Optional[str] = None

# ---------- Search ----------
class VectorSearchRequest(NamespaceRequest):
    vector: List[float]
    k: int = Field(..., gt=0, description="Number of results to return")
    normalize: bool = True

class SearchResult(BaseModel):
    id: int
    score: float

class VectorSearchResponse(BaseModel):
    ok: bool
    namespace: str
    k: int
    results: List[SearchResult] = Field(default_factory=list)
    error: Optional[str] = None

# ---------- Generic Responses ----------
class StandardResponse(BaseModel):
    ok: bool
    namespace: Optional[str] = None
    message: Optional[str] = None
    error: Optional[str] = None

class SystemClearResponse(BaseModel):
    ok: bool
    deleted_namespaces: int
    errors: List[str] = Field(default_factory=list)
